package gojinn

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func (r *Gojinn) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	start := time.Now()

	if err := r.handleMiddleware(rw, req); err != nil {
		return err
	}

	r.metrics.active.WithLabelValues(r.Path).Inc()
	defer r.metrics.active.WithLabelValues(r.Path).Dec()

	var pair *EnginePair
	select {
	case pair = <-r.enginePool:
	case <-time.After(time.Duration(r.Timeout)):
		return caddyhttp.Error(http.StatusServiceUnavailable, fmt.Errorf("worker pool exhausted"))
	}

	defer func() {
		select {
		case r.enginePool <- pair:
		default:
			pair.Runtime.Close(context.Background())
		}
	}()

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(r.Timeout))
	defer cancel()

	stdoutBuf := bufferPool.Get().(*bytes.Buffer)
	stdoutBuf.Reset()
	defer bufferPool.Put(stdoutBuf)

	stderrBuf := bufferPool.Get().(*bytes.Buffer)
	stderrBuf.Reset()
	defer bufferPool.Put(stderrBuf)

	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body.Close()

	reqPayload := struct {
		Method  string              `json:"method"`
		URI     string              `json:"uri"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	}{
		Method:  req.Method,
		URI:     req.RequestURI,
		Headers: req.Header,
		Body:    string(bodyBytes),
	}
	inputJSON, _ := json.Marshal(reqPayload)

	fsConfig := wazero.NewFSConfig()
	for host, guest := range r.Mounts {
		fsConfig = fsConfig.WithDirMount(host, guest)
	}

	modConfig := wazero.NewModuleConfig().
		WithStdout(stdoutBuf).
		WithStderr(stderrBuf).
		WithStdin(bytes.NewReader(inputJSON)).
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(rand.Reader).
		WithFSConfig(fsConfig)

	for k, v := range r.Env {
		modConfig = modConfig.WithEnv(k, v)
	}

	httpCtx := &HttpContext{
		W: rw,
		R: req,
	}
	ctxWithHTTP := context.WithValue(ctx, wsContextKey{}, httpCtx)

	mod, err := pair.Runtime.InstantiateModule(ctxWithHTTP, pair.Code, modConfig)
	if err != nil {
		if ctxWithHTTP.Err() == context.Canceled {
			return nil
		}
		r.logger.Error("WASM Execution Failed", zap.Error(err), zap.String("stderr", stderrBuf.String()))
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}
	defer mod.Close(ctxWithHTTP)

	if httpCtx.WSConn != nil {
		return nil
	}

	handleFunc := mod.ExportedFunction("handle")
	if handleFunc != nil {
	}

	return r.writeResponse(rw, stdoutBuf.Bytes(), time.Since(start).Seconds())
}

func (r *Gojinn) handleMiddleware(rw http.ResponseWriter, req *http.Request) error {
	origin := req.Header.Get("Origin")
	if len(r.CorsOrigins) > 0 && origin != "" {
		allowed := false
		for _, o := range r.CorsOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			rw.Header().Set("Access-Control-Allow-Origin", origin)
			rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			rw.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Gojinn-Debug, traceparent")
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if req.Method == "OPTIONS" {
			rw.WriteHeader(http.StatusOK)
			return nil
		}
	}
	tenantID := ""
	if len(r.APIKeys) > 0 {
		clientKey := req.Header.Get("X-API-Key")
		if clientKey == "" {
			authHeader := req.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				clientKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		authorized := false
		for _, k := range r.APIKeys {
			if clientKey == k {
				authorized = true
				break
			}
		}
		if !authorized {
			return caddyhttp.Error(http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		}
		tenantID = clientKey
	} else {
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		tenantID = host
	}
	if r.RateLimit > 0 {
		limiter := r.getLimiter(tenantID)
		if !limiter.Allow() {
			rw.WriteHeader(http.StatusTooManyRequests)
			return fmt.Errorf("rate limit exceeded")
		}
	}
	return nil
}

func (r *Gojinn) writeResponse(rw http.ResponseWriter, outBytes []byte, duration float64) error {
	if len(outBytes) == 0 {
		return caddyhttp.Error(http.StatusBadGateway, fmt.Errorf("empty response from wasm"))
	}
	var resp struct {
		Status  int                 `json:"status"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	}
	if err := json.Unmarshal(outBytes, &resp); err != nil {
		rw.Write(outBytes)
		return nil
	}
	for k, v := range resp.Headers {
		for _, val := range v {
			rw.Header().Add(k, val)
		}
	}
	if resp.Status == 0 {
		resp.Status = 200
	}
	rw.WriteHeader(resp.Status)
	rw.Write([]byte(resp.Body))
	statusLabel := fmt.Sprintf("%d", resp.Status)
	r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
	return nil
}
