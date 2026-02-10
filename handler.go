package gojinn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

type CrashSnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Error     string            `json:"error"`
	Input     json.RawMessage   `json:"input"`
	Env       map[string]string `json:"env"`
	WasmFile  string            `json:"wasm_file"`
}

var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func (r *Gojinn) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	if strings.HasPrefix(req.URL.Path, "/_sys/") {
		if req.URL.Path == "/_sys/status" {
			status := map[string]interface{}{
				"node_id":      "local-node",
				"uptime":       "running",
				"pool_size":    r.PoolSize,
				"memory_limit": r.MemoryLimit,
				"fuel_limit":   r.FuelLimit,
				"nats_status":  "disconnected",
				"topic":        r.getFunctionTopic(),
			}

			if r.natsConn != nil {
				status["nats_status"] = r.natsConn.Status().String()
			}

			rw.Header().Set("Content-Type", "application/json")
			json.NewEncoder(rw).Encode(status)
			return nil
		}

		if req.Method == "POST" && req.URL.Path == "/_sys/patch" {
			var patch struct {
				PoolSize int  `json:"pool_size"`
				Reload   bool `json:"reload"`
			}
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				http.Error(rw, err.Error(), 400)
				return nil
			}

			shouldReload := patch.Reload

			if patch.PoolSize > 0 {
				r.logger.Info("Hot Patching Pool Size",
					zap.Int("old_pool_size", r.PoolSize),
					zap.Int("new_pool_size", patch.PoolSize))

				r.PoolSize = patch.PoolSize
				shouldReload = true
			}

			if shouldReload {
				if err := r.ReloadWorkers(); err != nil {
					r.logger.Error("Hot Reload Failed", zap.Error(err))
					http.Error(rw, "Reload failed: "+err.Error(), 500)
					return nil
				}
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.Write([]byte(`{"status": "patched", "msg": "Configuration updated and workers reloaded hot!"}`))
			return nil
		}
	}

	start := time.Now()

	if err := r.handleMiddleware(rw, req); err != nil {
		return err
	}

	if r.metrics != nil {
		r.metrics.active.WithLabelValues(r.Path).Inc()
		defer r.metrics.active.WithLabelValues(r.Path).Dec()
	}

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

	topic := r.getFunctionTopic()

	natsTimeout := time.Duration(r.Timeout) + (100 * time.Millisecond)

	if r.natsConn == nil {
		return caddyhttp.Error(http.StatusServiceUnavailable, fmt.Errorf("NATS connection not available"))
	}

	msg, err := r.natsConn.Request(topic, inputJSON, natsTimeout)
	if err != nil {
		r.logger.Error("Function Execution Failed (NATS)", zap.Error(err))

		if r.RecordCrashes {
			snapshot := CrashSnapshot{
				Timestamp: time.Now(),
				Error:     err.Error(),
				Input:     inputJSON,
				Env:       r.Env,
				WasmFile:  r.Path,
			}
			dumpBytes, _ := json.MarshalIndent(snapshot, "", "  ")
			filename := fmt.Sprintf("crash_%d.json", time.Now().UnixNano())
			r.saveCrashDump(filename, dumpBytes)
		}

		return caddyhttp.Error(http.StatusServiceUnavailable, fmt.Errorf("worker execution failed: %v", err))
	}

	return r.writeResponse(rw, msg.Data, time.Since(start).Seconds())
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

	if r.metrics != nil {
		statusLabel := fmt.Sprintf("%d", resp.Status)
		r.metrics.duration.WithLabelValues(r.Path, statusLabel).Observe(duration)
	}
	return nil
}
