package gojinn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
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
	if strings.HasPrefix(req.URL.Path, "/mcp") {
		if req.URL.Path == "/mcp" || req.URL.Path == "/mcp/" {
			r.ServeMCP(rw, req)
			return nil
		}
		if req.URL.Path == "/mcp/message" {
			r.HandleMCPMessage(rw, req)
			return nil
		}
	}

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
			if err := json.NewEncoder(rw).Encode(status); err != nil {
				r.logger.Error("Failed to encode status", zap.Error(err))
			}
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
			if _, err := rw.Write([]byte(`{"status": "patched", "msg": "Configuration updated and workers reloaded hot!"}`)); err != nil {
				r.logger.Error("Failed to write response", zap.Error(err))
			}
			return nil
		}

		if req.Method == "POST" && req.URL.Path == "/_sys/snapshot" {
			snapshotPath, err := r.CreateGlobalSnapshot()

			rw.Header().Set("Content-Type", "application/json")
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(rw).Encode(map[string]string{
					"status": "error",
					"error":  err.Error(),
				})
				return nil
			}

			_ = json.NewEncoder(rw).Encode(map[string]string{
				"status": "success",
				"msg":    "Global Snapshot generated successfully",
				"file":   snapshotPath,
			})
			return nil
		}
	}

	if req.Method == "POST" && req.URL.Path == "/_sys/restore" {
		var payload struct {
			File string `json:"file"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(rw, "Invalid JSON payload", 400)
			return nil
		}

		if payload.File == "" {
			http.Error(rw, "Missing 'file' parameter", 400)
			return nil
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"status": "success",
			"msg":    "Restore accepted. System is overwriting disks and will restart.",
		})

		go func() {
			time.Sleep(500 * time.Millisecond)

			err := r.RestoreGlobalSnapshot(payload.File)
			if err != nil {
				r.logger.Fatal("Restore completely failed! Server is in an unknown state.", zap.Error(err))
			}

			os.Exit(0)
		}()

		return nil
	}

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

	if r.js == nil {
		return caddyhttp.Error(http.StatusServiceUnavailable, fmt.Errorf("JetStream not ready"))
	}

	pubAck, err := r.js.Publish(topic, inputJSON, nats.MsgId(fmt.Sprintf("%d", time.Now().UnixNano())))

	if err != nil {
		r.logger.Error("Failed to Persist Job (JetStream)", zap.Error(err))
		return caddyhttp.Error(http.StatusInternalServerError, fmt.Errorf("persistence failed: %v", err))
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Gojinn-Job-ID", fmt.Sprintf("%d", pubAck.Sequence))
	rw.WriteHeader(http.StatusAccepted)

	resp := map[string]interface{}{
		"status": "queued",
		"job_id": pubAck.Sequence,
		"stream": pubAck.Stream,
		"msg":    "Job persisted to disk and queued for execution.",
	}

	return json.NewEncoder(rw).Encode(resp)
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
