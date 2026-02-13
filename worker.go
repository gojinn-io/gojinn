package gojinn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	MaxRetries = 5
)

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

	fsConfig := wazero.NewFSConfig()
	for host, guest := range r.Mounts {
		fsConfig = fsConfig.WithDirMount(host, guest)
	}

	modConfig := wazero.NewModuleConfig().
		WithStdout(stdout).
		WithStderr(stderr).
		WithStdin(strings.NewReader(input)).
		WithSysWalltime().
		WithSysNanotime().
		WithFSConfig(fsConfig)

	for k, v := range r.Env {
		modConfig = modConfig.WithEnv(k, v)
	}

	mod, err := pair.Runtime.InstantiateModule(ctx, pair.Code, modConfig)
	if err != nil {
		return "", fmt.Errorf("wasm sync execution failed: %w | stderr: %s", err, stderr.String())
	}
	defer mod.Close(ctx)

	return stdout.String(), nil
}

func (r *Gojinn) startWorkerSubscriber(id int, topic string, wasmBytes []byte) (*nats.Subscription, error) {
	pair, err := r.createWazeroRuntime(wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create wazero runtime for worker %d: %w", id, err)
	}

	queueGroup := fmt.Sprintf("WORKERS_%s", hashString(r.Path))

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

		fsConfig := wazero.NewFSConfig()
		for host, guest := range r.Mounts {
			fsConfig = fsConfig.WithDirMount(host, guest)
		}

		modConfig := wazero.NewModuleConfig().
			WithStdout(stdoutBuf).
			WithStderr(stderrBuf).
			WithStdin(bytes.NewReader(m.Data)).
			WithSysWalltime().
			WithSysNanotime().
			WithFSConfig(fsConfig)

		for k, v := range r.Env {
			modConfig = modConfig.WithEnv(k, v)
		}

		mod, err := pair.Runtime.InstantiateModule(ctx, pair.Code, modConfig)
		if err != nil {
			errMsg := fmt.Sprintf("Wasm Error: %v | Stderr: %s", err, stderrBuf.String())

			if deliverCount >= MaxRetries {
				snapshot := CrashSnapshot{
					Timestamp: time.Now(),
					Error:     errMsg,
					Input:     json.RawMessage(m.Data),
					Env:       r.Env,
					WasmFile:  r.Path,
				}
				dumpBytes, _ := json.MarshalIndent(snapshot, "", "  ")
				filename := fmt.Sprintf("crash_%s_seq%d.json", time.Now().Format("20060102-150405"), meta.Sequence.Stream)
				r.saveCrashDump(filename, dumpBytes)
				_ = m.Ack()
				return
			}

			backoff := time.Duration(deliverCount) * time.Second
			_ = m.NakWithDelay(backoff)
			return
		}

		mod.Close(ctx)
		_ = m.Ack()

	}, nats.ManualAck(), nats.BindStream("GOJINN_WORKER"), nats.MaxDeliver(MaxRetries+1))

	return sub, err
}
