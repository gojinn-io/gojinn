package gojinn

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"nhooyr.io/websocket"

	"github.com/pauloappbr/gojinn/pkg/mesh"
	"github.com/pauloappbr/gojinn/pkg/sovereign"
)

func init() {
	caddy.RegisterModule(&Gojinn{})
	httpcaddyfile.RegisterHandlerDirective("gojinn", parseCaddyfile)
}

type Gojinn struct {
	Path        string            `json:"path,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     caddy.Duration    `json:"timeout,omitempty"`
	MemoryLimit string            `json:"memory_limit,omitempty"`
	PoolSize    int               `json:"pool_size,omitempty"`
	DebugSecret string            `json:"debug_secret,omitempty"`

	TrustedKeys    []string `json:"trusted_keys,omitempty"`
	SecurityPolicy string   `json:"security_policy,omitempty"`

	ClusterPort   int      `json:"cluster_port,omitempty"`
	ClusterSeeds  []string `json:"cluster_seeds,omitempty"`
	ClusterSecret string   `json:"cluster_secret,omitempty"`
	meshNode      *mesh.Node

	FuelLimit uint64            `json:"fuel_limit,omitempty"`
	Mounts    map[string]string `json:"mounts,omitempty"`

	DBDriver string `json:"db_driver,omitempty"`
	DBDSN    string `json:"db_dsn,omitempty"`
	kvStore  sync.Map

	db      *sql.DB
	logger  *zap.Logger
	metrics *gojinnMetrics

	enginePool chan *EnginePair

	S3Endpoint  string `json:"s3_endpoint,omitempty"`
	S3Region    string `json:"s3_region,omitempty"`
	S3Bucket    string `json:"s3_bucket,omitempty"`
	S3AccessKey string `json:"s3_access_key,omitempty"`
	S3SecretKey string `json:"s3_secret_key,omitempty"`

	CronJobs  []CronJob `json:"cron_jobs,omitempty"`
	scheduler *cron.Cron

	MQTTBroker   string    `json:"mqtt_broker,omitempty"`
	MQTTClientID string    `json:"mqtt_client_id,omitempty"`
	MQTTUsername string    `json:"mqtt_username,omitempty"`
	MQTTPassword string    `json:"mqtt_password,omitempty"`
	MQTTSubs     []MQTTSub `json:"mqtt_subs,omitempty"`
	mqttClient   mqtt.Client

	AIProvider string `json:"ai_provider,omitempty"`
	AIModel    string `json:"ai_model,omitempty"`
	AIEndpoint string `json:"ai_endpoint,omitempty"`
	AIToken    string `json:"ai_token,omitempty"`
	aiCache    sync.Map

	APIKeys      []string `json:"api_keys,omitempty"`
	AllowedHosts []string `json:"allowed_hosts,omitempty"`
	CorsOrigins  []string `json:"cors_origins,omitempty"`

	RateLimit  float64 `json:"rate_limit,omitempty"`
	RateBurst  int     `json:"rate_burst,omitempty"`
	limiters   map[string]*rate.Limiter
	limitersMu sync.Mutex
}

func (*Gojinn) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.gojinn",
		New: func() caddy.Module { return &Gojinn{} },
	}
}

func (g *Gojinn) loadWasmSecurely(path string) ([]byte, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	g.logger.Debug("LoadWasm Raw", zap.String("file", path), zap.Int("size", len(rawBytes)))

	if len(g.TrustedKeys) == 0 {
		if g.SecurityPolicy == "strict" {
			return nil, fmt.Errorf("security policy is strict but no trusted keys are defined")
		}
		return rawBytes, nil
	}

	var trusted []ed25519.PublicKey
	for _, k := range g.TrustedKeys {
		pk, err := sovereign.ParsePublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted key config: %w", err)
		}
		trusted = append(trusted, pk)
	}

	cleanBytes, err := sovereign.VerifyWasm(rawBytes, trusted)
	if err != nil {
		if g.SecurityPolicy == "strict" {
			g.logger.Error("BLOCKING UNSIGNED MODULE", zap.String("file", path), zap.Error(err))
			return nil, fmt.Errorf("module signature verification failed: %w", err)
		}
		g.logger.Warn("Security Audit Failed (Allowing run due to audit policy)",
			zap.String("file", path),
			zap.Error(err))
		return rawBytes, nil
	}

	g.logger.Info("Module Signature Verified", zap.String("file", path), zap.Int("size_clean", len(cleanBytes)))
	return cleanBytes, nil
}

func (r *Gojinn) Provision(ctx caddy.Context) error {
	r.logger = ctx.Logger()
	r.limiters = make(map[string]*rate.Limiter)

	if err := r.setupMetrics(ctx); err != nil {
		return err
	}
	if err := r.setupDB(); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}

	if r.ClusterPort > 0 {
		r.logger.Info("Initializing Mesh Node", zap.Int("port", r.ClusterPort))

		node, err := mesh.NewNode(r.ClusterPort, r.ClusterPort, r.ClusterSeeds, r.ClusterSecret, []string{r.Path}, 8080)
		if err != nil {
			return fmt.Errorf("failed to start mesh: %w", err)
		}
		r.meshNode = node

		r.meshNode.OnKVUpdate = func(key, value string) {
			r.logger.Debug("Mesh KV Sync (Remote Update)", zap.String("key", key), zap.String("val_size", fmt.Sprintf("%d bytes", len(value))))
			r.kvStore.Store(key, value)
		}

		go func() {
			time.Sleep(2 * time.Second)
			ticker := time.NewTicker(30 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if r.meshNode != nil && r.meshNode.List != nil {
						peers := r.meshNode.GetPeers()
						r.logger.Info("Mesh Topology",
							zap.Int("members_count", len(peers)),
							zap.Strings("members", peers))
					}
				}
			}
		}()
	}

	if len(r.CronJobs) > 0 {
		r.scheduler = cron.New(cron.WithSeconds())
		for _, job := range r.CronJobs {
			j := job

			if _, err := r.loadWasmSecurely(j.WasmFile); err != nil {
				return fmt.Errorf("cron job security check failed for %s: %w", j.WasmFile, err)
			}

			_, err := r.scheduler.AddFunc(j.Schedule, func() {
				r.runBackgroundJob(j.WasmFile)
			})
			if err != nil {
				return fmt.Errorf("failed to schedule cron job: %v", err)
			}
			r.logger.Info("Cron job scheduled", zap.String("schedule", j.Schedule), zap.String("wasm", j.WasmFile))
		}
		r.scheduler.Start()
	}

	if r.MQTTBroker != "" {
		for _, sub := range r.MQTTSubs {
			if _, err := r.loadWasmSecurely(sub.WasmFile); err != nil {
				return fmt.Errorf("mqtt handler security check failed for %s: %w", sub.WasmFile, err)
			}
		}

		opts := mqtt.NewClientOptions()
		opts.AddBroker(r.MQTTBroker)
		if r.MQTTClientID != "" {
			opts.SetClientID(r.MQTTClientID)
		}
		if r.MQTTUsername != "" {
			opts.SetUsername(r.MQTTUsername)
		}
		if r.MQTTPassword != "" {
			opts.SetPassword(r.MQTTPassword)
		}

		opts.OnConnect = func(c mqtt.Client) {
			r.logger.Info("MQTT Connected", zap.String("broker", r.MQTTBroker))
			for _, sub := range r.MQTTSubs {
				s := sub
				token := c.Subscribe(s.Topic, 0, func(client mqtt.Client, msg mqtt.Message) {
					go r.runAsyncJob(s.WasmFile, string(msg.Payload()))
				})
				if token.Wait() && token.Error() != nil {
					r.logger.Error("MQTT Subscribe Error", zap.Error(token.Error()))
				} else {
					r.logger.Info("MQTT Subscribed", zap.String("topic", s.Topic))
				}
			}
		}
		opts.OnConnectionLost = func(c mqtt.Client, err error) {
			r.logger.Warn("MQTT Connection Lost", zap.Error(err))
		}
		r.mqttClient = mqtt.NewClient(opts)
		if token := r.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			return fmt.Errorf("MQTT connect error: %w", token.Error())
		}
	}

	if r.PoolSize <= 0 {
		r.PoolSize = 2
	}
	if r.Timeout == 0 {
		r.Timeout = caddy.Duration(60 * time.Second)
	}

	if r.Path != "" {
		r.enginePool = make(chan *EnginePair, r.PoolSize)

		wasmBytes, err := r.loadWasmSecurely(r.Path)
		if err != nil {
			return fmt.Errorf("failed to load sovereign module: %w", err)
		}
		r.logger.Info("provisioning worker pool", zap.Int("workers", r.PoolSize))

		var wg sync.WaitGroup
		for i := 0; i < r.PoolSize; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pair, err := r.createWorker(wasmBytes)
				if err != nil {
					r.logger.Error("failed to provision worker", zap.Error(err))
					return
				}
				r.enginePool <- pair
			}()
		}
		wg.Wait()
	}

	return nil
}

func (r *Gojinn) Cleanup() error {
	if r.meshNode != nil {
		r.logger.Info("Leaving Mesh Cluster...")
		r.meshNode.Shutdown()
	}
	if r.mqttClient != nil && r.mqttClient.IsConnected() {
		r.mqttClient.Disconnect(250)
	}
	if r.scheduler != nil {
		r.scheduler.Stop()
	}
	if r.db != nil {
		r.db.Close()
	}
	if r.enginePool != nil {
		close(r.enginePool)
	}
	return nil
}

func (r *Gojinn) getLimiter(key string) *rate.Limiter {
	r.limitersMu.Lock()
	defer r.limitersMu.Unlock()
	limiter, exists := r.limiters[key]
	if !exists {
		burst := r.RateBurst
		if burst == 0 {
			burst = int(r.RateLimit)
		}
		if burst == 0 {
			burst = 1
		}
		limiter = rate.NewLimiter(rate.Limit(r.RateLimit), burst)
		r.limiters[key] = limiter
	}
	return limiter
}

type AIRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
	Stream   bool        `json:"stream"`
}
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type AIResponse struct {
	Choices []struct {
		Message AIMessage `json:"message"`
	} `json:"choices"`
}

func (g *Gojinn) askAI(prompt string) (string, error) {
	provider := g.AIProvider
	if provider == "" {
		provider = "openai"
	}
	model := g.AIModel
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	cacheKey := fmt.Sprintf("%s:%s", model, hashString(prompt))
	if cachedVal, ok := g.aiCache.Load(cacheKey); ok {
		return cachedVal.(string), nil
	}

	endpoint := g.AIEndpoint
	if endpoint == "" {
		if provider == "ollama" {
			endpoint = "http://localhost:11434/v1/chat/completions"
		} else {
			endpoint = "https://api.openai.com/v1/chat/completions"
		}
	}
	if len(g.AllowedHosts) > 0 {
		u, err := url.Parse(endpoint)
		if err == nil {
			allowed := false
			hostname := u.Hostname()
			if provider == "ollama" && (hostname == "localhost" || hostname == "127.0.0.1") {
				allowed = true
			} else {
				for _, host := range g.AllowedHosts {
					if strings.Contains(hostname, host) {
						allowed = true
						break
					}
				}
			}
			if !allowed {
				return "", fmt.Errorf("egress denied to %s", hostname)
			}
		}
	}

	reqBody := AIRequest{
		Model:  model,
		Stream: false,
		Messages: []AIMessage{
			{Role: "system", Content: "You are a helpful assistant running inside Gojinn Serverless."},
			{Role: "user", Content: prompt},
		},
	}
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if g.AIToken != "" {
		req.Header.Set("Authorization", "Bearer "+g.AIToken)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI connect error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI API error (%d): %s", resp.StatusCode, string(body))
	}

	var aiResp AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", fmt.Errorf("json decode error: %w", err)
	}

	if len(aiResp.Choices) > 0 {
		responseContent := aiResp.Choices[0].Message.Content
		g.aiCache.Store(cacheKey, responseContent)
		return responseContent, nil
	}
	return "", fmt.Errorf("AI returned no response")
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

type wsContextKey struct{}

type HttpContext struct {
	W      http.ResponseWriter
	R      *http.Request
	WSConn *websocket.Conn
}
