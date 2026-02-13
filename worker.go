package gojinn

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	MaxRetries     = 5
	MaxOutputBytes = 5 * 1024 * 1024
)

type cappedWriter struct {
	buf     *bytes.Buffer
	limit   int
	written int
	cancel  context.CancelFunc
}

func (cw *cappedWriter) Write(p []byte) (int, error) {
	if cw.written+len(p) > cw.limit {
		allowed := cw.limit - cw.written
		if allowed > 0 {
			cw.buf.Write(p[:allowed])
			cw.written += allowed
		}

		if cw.cancel != nil {
			cw.cancel()
		}
		return 0, fmt.Errorf("tenant output quota exceeded (max %d bytes)", cw.limit)
	}
	n, err := cw.buf.Write(p)
	cw.written += n
	return n, err
}

func (r *Gojinn) runSyncJob(ctx context.Context, wasmPath string, input string) (string, error) {
	wasmBytes, err := r.loadWasmSecurely(wasmPath)
	if err != nil {
		return "", err
	}

	pair, err := r.createWazeroRuntime(wasmBytes)
	if err != nil {
		return "", err
	}
	defer pair.Runtime.Close(ctx)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cwOut := &cappedWriter{buf: stdout, limit: MaxOutputBytes, cancel: cancel}
	cwErr := &cappedWriter{buf: stderr, limit: MaxOutputBytes, cancel: cancel}

	fsConfig := wazero.NewFSConfig()
	for host, guest := range r.Mounts {
		fsConfig = fsConfig.WithDirMount(host, guest)
	}

	modConfig := wazero.NewModuleConfig().
		WithStdout(cwOut).
		WithStderr(cwErr).
		WithStdin(strings.NewReader(input)).
		WithSysWalltime().
		WithSysNanotime().
		WithFSConfig(fsConfig)

	for k, v := range r.Env {
		modConfig = modConfig.WithEnv(k, v)
	}

	mod, err := pair.Runtime.InstantiateModule(execCtx, pair.Code, modConfig)
	if err != nil {
		return "", fmt.Errorf("wasm sync execution failed: %w | stderr: %s", err, stderr.String())
	}
	defer mod.Close(execCtx)

	return stdout.String(), nil
}

func (r *Gojinn) startTenantWorker(tenantID string, streamName string, id int, topic string, wasmBytes []byte) (*nats.Subscription, error) {
	pair, err := r.createWazeroRuntime(wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero runtime for tenant %s worker %d: %w", tenantID, id, err)
	}

	queueGroup := fmt.Sprintf("WORKERS_%s", tenantID)

	sub, err := r.js.QueueSubscribe(topic, queueGroup, func(m *nats.Msg) {
		meta, err := m.Metadata()
		if err != nil {
			r.logger.Error("Failed to get msg metadata", zap.Error(err))
			_ = m.Nak()
			return
		}

		deliverCount := meta.NumDelivered
		_ = m.InProgress()

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Timeout))
		defer cancel()

		stdoutBuf := bufferPool.Get().(*bytes.Buffer)
		stdoutBuf.Reset()
		defer bufferPool.Put(stdoutBuf)

		stderrBuf := bufferPool.Get().(*bytes.Buffer)
		stderrBuf.Reset()
		defer bufferPool.Put(stderrBuf)

		cwOut := &cappedWriter{buf: stdoutBuf, limit: MaxOutputBytes, cancel: cancel}
		cwErr := &cappedWriter{buf: stderrBuf, limit: MaxOutputBytes, cancel: cancel}

		fsConfig := wazero.NewFSConfig()
		for host, guest := range r.Mounts {
			fsConfig = fsConfig.WithDirMount(host, guest)
		}

		modConfig := wazero.NewModuleConfig().
			WithStdout(cwOut).
			WithStderr(cwErr).
			WithStdin(bytes.NewReader(m.Data)).
			WithSysWalltime().
			WithSysNanotime().
			WithFSConfig(fsConfig)

		for k, v := range r.Env {
			modConfig = modConfig.WithEnv(k, v)
		}

		mod, err := pair.Runtime.InstantiateModule(ctx, pair.Code, modConfig)
		if err != nil {
			errMsg := fmt.Sprintf("Wasm Error/Quota Exceeded: %v | Stderr: %s", err, stderrBuf.String())

			if deliverCount >= MaxRetries {
				snapshot := CrashSnapshot{
					Timestamp: time.Now(),
					Error:     errMsg,
					Input:     json.RawMessage(m.Data),
					Env:       r.Env,
					WasmFile:  r.Path,
				}
				dumpBytes, _ := json.MarshalIndent(snapshot, "", "  ")
				filename := fmt.Sprintf("crash_tenant_%s_%s_seq%d.json", tenantID, time.Now().Format("20060102-150405"), meta.Sequence.Stream)
				r.saveCrashDump(filename, dumpBytes)
				_ = m.Ack()
				return
			}

			backoff := time.Duration(deliverCount) * time.Second
			_ = m.NakWithDelay(backoff)
			return
		}

		if stdoutBuf.Len() > 0 {
			r.logger.Info("Tenant Worker Output", zap.String("tenant", tenantID), zap.String("stdout", strings.TrimSpace(stdoutBuf.String())))
		}
		if stderrBuf.Len() > 0 {
			r.logger.Info("Tenant Worker Log", zap.String("tenant", tenantID), zap.String("stderr", strings.TrimSpace(stderrBuf.String())))
		}

		kvBucket := fmt.Sprintf("STATE_%s", strings.ToUpper(tenantID))
		kv, kvErr := r.js.KeyValue(kvBucket)
		if kvErr == nil {
			outStr := strings.TrimSpace(stdoutBuf.String())
			errStr := strings.TrimSpace(stderrBuf.String())
			timestamp := time.Now().UTC().Format(time.RFC3339)

			payload := fmt.Sprintf("tenant:%s|job:%d|out:%s|err:%s|ts:%s", tenantID, meta.Sequence.Stream, outStr, errStr, timestamp)

			secret := r.StoreCipherKey
			if secret == "" {
				secret = "gojinn-default-audit-secret"
			}
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(payload))
			signature := hex.EncodeToString(mac.Sum(nil))

			auditData := map[string]interface{}{
				"job_id":    meta.Sequence.Stream,
				"timestamp": timestamp,
				"signature": signature,
				"status":    "success",
			}
			auditJSON, _ := json.Marshal(auditData)

			auditKey := fmt.Sprintf("audit.job.%d", meta.Sequence.Stream)
			_, _ = kv.Put(auditKey, auditJSON)

			r.logger.Info("Signed Audit Log Saved", zap.String("tenant", tenantID), zap.String("audit_key", auditKey), zap.String("signature", signature[:16]+"..."))
		}

		mod.Close(ctx)
		_ = m.Ack()

	}, nats.ManualAck(), nats.BindStream(streamName), nats.MaxDeliver(MaxRetries+1))

	return sub, err
}
