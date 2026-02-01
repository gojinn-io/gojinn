package gojinn

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (r *Gojinn) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	start := time.Now()
	r.metrics.active.WithLabelValues(r.Path).Inc()
	defer r.metrics.active.WithLabelValues(r.Path).Dec()

	pair := <-r.enginePool
	defer func() { r.enginePool <- pair }()

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(r.Timeout))
	defer cancel()

	isDebug := r.DebugSecret != "" && req.Header.Get("X-Gojinn-Debug") == r.DebugSecret

	stderrBuf := bufferPool.Get().(*bytes.Buffer)
	stderrBuf.Reset()
	defer bufferPool.Put(stderrBuf)

	var stderrTarget io.Writer = io.MultiWriter(os.Stderr, stderrBuf)
	var debugLogBuf *bytes.Buffer

	if isDebug {
		debugLogBuf = bufferPool.Get().(*bytes.Buffer)
		debugLogBuf.Reset()
		defer bufferPool.Put(debugLogBuf)

		stderrTarget = io.MultiWriter(os.Stderr, debugLogBuf)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	traceID := req.Header.Get("traceparent")

	reqPayload := struct {
		Method  string              `json:"method"`
		URI     string              `json:"uri"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
		TraceID string              `json:"trace_id,omitempty"`
	}{
		Method:  req.Method,
		URI:     req.RequestURI,
		Headers: req.Header,
		Body:    string(bodyBytes),
		TraceID: traceID,
	}

	inputJSON, err := json.Marshal(reqPayload)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	stdoutBuf := bufferPool.Get().(*bytes.Buffer)
	stdoutBuf.Reset()
	defer bufferPool.Put(stdoutBuf)

	fullArgs := append([]string{"python"}, r.Args...)

	config := wazero.NewModuleConfig().
		WithStdout(stdoutBuf).
		WithStderr(stderrTarget).
		WithStdin(bytes.NewReader(inputJSON)).
		WithArgs(fullArgs...).
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(rand.Reader).
		WithFSConfig(wazero.NewFSConfig().WithDirMount(".", "/"))

	for k, v := range r.Env {
		config = config.WithEnv(k, v)
	}

	instance, err := pair.Runtime.InstantiateModule(ctx, pair.Code, config)

	duration := time.Since(start).Seconds()
	var statusLabel string

	injectDebugLogs := func() {
		if isDebug && debugLogBuf.Len() > 0 {
			logs := debugLogBuf.String()
			if len(logs) > 4096 {
				logs = logs[:4096] + "...(truncated)"
			}
			rw.Header().Set("X-Gojinn-Logs", logs)
		}
	}

	if err != nil {
		statusLabel = "500"
		if ctx.Err() == context.DeadlineExceeded {
			statusLabel = "504"
		}
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)

		r.logger.Error("wasm execution failed",
			zap.Error(err),
			zap.String("stderr_preview", stderrBuf.String()),
		)

		injectDebugLogs()
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}
	defer instance.Close(ctx)

	if stdoutBuf.Len() == 0 {
		statusLabel = "502"
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)

		injectDebugLogs()

		jsError := stderrBuf.String()
		if jsError == "" {
			jsError = "No stderr output. (Possible causes: script not found, early exit)"
		}

		r.logger.Error("WASM Empty Output", zap.String("stderr", jsError))

		return caddyhttp.Error(http.StatusBadGateway, fmt.Errorf("empty response from WASM. Stderr: %s", jsError))
	}

	var respPayload struct {
		Status  int                    `json:"status"`
		Headers map[string]interface{} `json:"headers"`
		Body    string                 `json:"body"`
	}

	if err := json.Unmarshal(stdoutBuf.Bytes(), &respPayload); err != nil {
		statusLabel = "502"
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)

		rawOutput := stdoutBuf.String()
		r.logger.Error("Invalid JSON from WASM", zap.String("output", rawOutput), zap.Error(err))

		return caddyhttp.Error(http.StatusBadGateway, fmt.Errorf("invalid json output: %s. Error: %v", rawOutput, err))
	}

	if respPayload.Status == 0 {
		respPayload.Status = 200
	}
	statusLabel = fmt.Sprintf("%d", respPayload.Status)
	r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)

	injectDebugLogs()

	for k, v := range respPayload.Headers {
		switch val := v.(type) {
		case string:
			rw.Header().Add(k, val)
		case []interface{}:
			for _, item := range val {
				if strVal, ok := item.(string); ok {
					rw.Header().Add(k, strVal)
				}
			}
		case []string:
			for _, strVal := range val {
				rw.Header().Add(k, strVal)
			}
		}
	}
	rw.WriteHeader(respPayload.Status)
	if _, err := rw.Write([]byte(respPayload.Body)); err != nil {
		r.logger.Error("failed to write response body", zap.Error(err))
	}

	return nil
}
