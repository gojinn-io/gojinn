package gojinn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/dustin/go-humanize"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func init() {
	caddy.RegisterModule(Gojinn{})
	httpcaddyfile.RegisterHandlerDirective("gojinn", parseCaddyfile)
}

// gojinnMetrics holds the initialized Prometheus metrics.
// We store them in the struct to handle Caddy reloads correctly.
type gojinnMetrics struct {
	duration *prometheus.HistogramVec
	active   *prometheus.GaugeVec
}

type Gojinn struct {
	Path        string            `json:"path,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     caddy.Duration    `json:"timeout,omitempty"`
	MemoryLimit string            `json:"memory_limit,omitempty"`

	logger  *zap.Logger
	code    wazero.CompiledModule
	engine  wazero.Runtime
	metrics *gojinnMetrics // Metrics are stored here, not globally
}

func (Gojinn) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.gojinn",
		New: func() caddy.Module { return &Gojinn{} },
	}
}

func (r *Gojinn) Provision(ctx caddy.Context) error {
	r.logger = ctx.Logger()

	// --- METRICS REGISTRATION (CADDY WAY) ---
	// Get Caddy's internal metrics registry to ensure scope isolation
	registry := ctx.GetMetricsRegistry()
	r.metrics = &gojinnMetrics{}

	// 1. Define Histogram
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gojinn_function_duration_seconds",
		Help:    "Time taken to execute the WASM function",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "status"})

	// Try to register. If already registered (e.g., during reload), recover the existing collector.
	if err := registry.Register(duration); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.duration = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			return fmt.Errorf("failed to register duration metric: %v", err)
		}
	} else {
		r.metrics.duration = duration
	}

	// 2. Define Gauge
	active := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gojinn_active_sandboxes",
		Help: "Number of WASM sandboxes currently running",
	}, []string{"path"})

	if err := registry.Register(active); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			r.metrics.active = are.ExistingCollector.(*prometheus.GaugeVec)
		} else {
			return fmt.Errorf("failed to register active metric: %v", err)
		}
	} else {
		r.metrics.active = active
	}
	// ---------------------------------------------------

	if r.Path == "" {
		return fmt.Errorf("wasm file path is required")
	}

	ctxWazero := context.Background()
	rConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)

	if r.MemoryLimit != "" {
		bytes, err := humanize.ParseBytes(r.MemoryLimit)
		if err != nil {
			return fmt.Errorf("invalid memory_limit: %v", err)
		}
		if bytes > 0 {
			const wasmPageSize = 65536
			pages := uint32(bytes / wasmPageSize)
			if bytes%wasmPageSize != 0 {
				pages++
			}
			rConfig = rConfig.WithMemoryLimitPages(pages)
		}
	}

	r.engine = wazero.NewRuntimeWithConfig(ctxWazero, rConfig)

	// Host Module for Logs
	// Allows guest WASM to call host_log(level, ptr, size)
	_, err := r.engine.NewHostModuleBuilder("gojinn").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			level := uint32(stack[0])
			ptr := uint32(stack[1])
			size := uint32(stack[2])

			msgBytes, ok := mod.Memory().Read(ptr, size)
			if !ok {
				return
			}
			msg := string(msgBytes)

			switch level {
			case 0:
				r.logger.Debug(msg, zap.String("source", "wasm"))
			case 1:
				r.logger.Info(msg, zap.String("source", "wasm"))
			case 2:
				r.logger.Warn(msg, zap.String("source", "wasm"))
			case 3:
				r.logger.Error(msg, zap.String("source", "wasm"))
			default:
				r.logger.Info(msg, zap.String("source", "wasm"))
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_log").
		Instantiate(ctxWazero)

	if err != nil {
		return fmt.Errorf("failed to instantiate host module: %w", err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctxWazero, r.engine)

	wasmBytes, err := os.ReadFile(r.Path)
	if err != nil {
		return fmt.Errorf("failed to read wasm file: %w", err)
	}

	r.code, err = r.engine.CompileModule(ctxWazero, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile wasm binary: %w", err)
	}

	if r.Timeout == 0 {
		r.Timeout = caddy.Duration(60 * time.Second)
	}

	return nil
}

func (r *Gojinn) Cleanup() error {
	if r.engine != nil {
		return r.engine.Close(context.Background())
	}
	return nil
}

func (r *Gojinn) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	start := time.Now()

	// Increment active sandboxes metric
	r.metrics.active.WithLabelValues(r.Path).Inc()
	defer r.metrics.active.WithLabelValues(r.Path).Dec()

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(r.Timeout))
	defer cancel()

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	// Distributed Tracing Injection
	traceID := req.Header.Get("traceparent")
	if traceID == "" {
		traceID = req.Header.Get("X-Request-Id")
	}

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
		r.logger.Error("failed to marshal request", zap.Error(err))
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	var stdoutBuf bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdoutBuf).
		WithStderr(os.Stderr).
		WithStdin(bytes.NewReader(inputJSON)).
		WithArgs(r.Args...)

	for k, v := range r.Env {
		config = config.WithEnv(k, v)
	}

	instance, err := r.engine.InstantiateModule(ctx, r.code, config)

	duration := time.Since(start).Seconds()
	statusLabel := "200"

	if err != nil {
		statusLabel = "500"
		if ctx.Err() == context.DeadlineExceeded {
			statusLabel = "504"
			// Record timeout metric
			r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
			return caddyhttp.Error(http.StatusGatewayTimeout, fmt.Errorf("execution time limit exceeded"))
		}
		// Record error metric
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
		r.logger.Error("wasm execution failed", zap.Error(err))
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}
	defer instance.Close(ctx)

	if stdoutBuf.Len() == 0 {
		statusLabel = "500"
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
		r.logger.Error("wasm returned empty response (crashed?)")
		return caddyhttp.Error(http.StatusInternalServerError, fmt.Errorf("wasm module crashed or returned no data"))
	}

	var respPayload struct {
		Status  int                 `json:"status"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	}

	if err := json.Unmarshal(stdoutBuf.Bytes(), &respPayload); err != nil {
		statusLabel = "502"
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
		r.logger.Error("invalid json response from wasm",
			zap.Error(err),
			zap.String("raw_output", stdoutBuf.String()))
		return caddyhttp.Error(http.StatusBadGateway, fmt.Errorf("wasm returned invalid protocol json"))
	}

	statusLabel = fmt.Sprintf("%d", respPayload.Status)
	if respPayload.Status == 0 {
		statusLabel = "200"
	}
	r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)

	for k, v := range respPayload.Headers {
		for _, val := range v {
			rw.Header().Add(k, val)
		}
	}

	if respPayload.Status == 0 {
		respPayload.Status = 200
	}

	rw.WriteHeader(respPayload.Status)
	rw.Write([]byte(respPayload.Body))

	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Gojinn
	m.Env = make(map[string]string)

	for h.Next() {
		args := h.RemainingArgs()
		if len(args) > 0 {
			m.Path = args[0]
		}

		for h.NextBlock(0) {
			switch h.Val() {
			case "env":
				if h.NextArg() {
					key := h.Val()
					if h.NextArg() {
						m.Env[key] = h.Val()
					}
				}
			case "args":
				m.Args = h.RemainingArgs()
			case "timeout":
				if h.NextArg() {
					val, err := caddy.ParseDuration(h.Val())
					if err != nil {
						return nil, h.Errf("invalid duration: %v", err)
					}
					m.Timeout = caddy.Duration(val)
				}
			case "memory_limit":
				if h.NextArg() {
					m.MemoryLimit = h.Val()
				}
			}
		}
	}
	return &m, nil
}
