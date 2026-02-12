package gojinn

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/coder/websocket"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
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

	RecordCrashes bool   `json:"record_crashes,omitempty"`
	CrashPath     string `json:"crash_path,omitempty"`

	DataDir string `json:"data_dir,omitempty"`

	TrustedKeys    []string `json:"trusted_keys,omitempty"`
	SecurityPolicy string   `json:"security_policy,omitempty"`

	NatsPort   int      `json:"nats_port,omitempty"`
	NatsRoutes []string `json:"nats_routes,omitempty"`
	natsServer *server.Server
	natsConn   *nats.Conn
	js         nats.JetStreamContext

	FuelLimit uint64            `json:"fuel_limit,omitempty"`
	Mounts    map[string]string `json:"mounts,omitempty"`

	DBDriver string `json:"db_driver,omitempty"`
	DBDSN    string `json:"db_dsn,omitempty"`
	kv       nats.KeyValue

	db      *sql.DB
	logger  *zap.Logger
	metrics *gojinnMetrics

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

	subs   []*nats.Subscription
	subsMu sync.Mutex

	ClusterName  string   `json:"cluster_name,omitempty"`
	ClusterPort  int      `json:"cluster_port,omitempty"`
	ClusterPeers []string `json:"cluster_peers,omitempty"`
	ServerName   string   `json:"server_name,omitempty"`

	LeafRemotes []string `json:"leaf_remotes,omitempty"`
	LeafPort    int      `json:"leaf_port,omitempty"`
}

func (*Gojinn) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.gojinn",
		New: func() caddy.Module { return &Gojinn{} },
	}
}

func (r *Gojinn) Provision(ctx caddy.Context) error {
	r.logger = ctx.Logger()

	shutdown, err := setupTelemetry("gojinn-" + r.ClusterName)
	if err != nil {
		r.logger.Warn("Failed to setup telemetry", zap.Error(err))
	} else {
		_ = shutdown
	}

	r.limiters = make(map[string]*rate.Limiter)

	if r.DataDir == "" {
		r.DataDir = "./data"
	}
	if err := os.MkdirAll(r.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := r.setupMetrics(ctx); err != nil {
		return err
	}
	if err := r.setupDB(); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}

	if r.ClusterName == "" {
		r.ClusterName = "gojinn-cluster"
	}

	if err := r.startEmbeddedNATS(); err != nil {
		return err
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
					go r.runAsyncJob(context.Background(), s.WasmFile, string(msg.Payload()))
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
			r.logger.Error("MQTT Initial Connect Failed", zap.Error(token.Error()))
		}
	}

	if r.PoolSize <= 0 {
		r.PoolSize = 2
	}
	if r.Timeout == 0 {
		r.Timeout = caddy.Duration(60 * time.Second)
	}

	if r.Path != "" {
		wasmBytes, err := r.loadWasmSecurely(r.Path)
		if err != nil {
			return fmt.Errorf("failed to load sovereign module: %w", err)
		}

		go r.startWorkersAsync(wasmBytes)
	}

	return nil
}

func (r *Gojinn) startWorkersAsync(wasmBytes []byte) {
	r.logger.Info("Provisioning NATS workers (Async)...", zap.Int("workers", r.PoolSize))
	topic := r.getFunctionTopic()
	streamName := "GOJINN_WORKER"

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if r.js == nil {
			continue
		}

		_, err := r.js.StreamInfo(streamName)
		if err != nil {
			continue
		}

		r.subsMu.Lock()
		for i := 0; i < r.PoolSize; i++ {
			sub, err := r.startWorkerSubscriber(i, topic, wasmBytes)
			if err != nil {
				r.logger.Error("Failed to start worker subscriber", zap.Error(err))
			} else {
				r.subs = append(r.subs, sub)
			}
		}
		r.subsMu.Unlock()

		r.logger.Info("All Workers Started Successfully via JetStream!", zap.Int("count", len(r.subs)))
		return
	}
}

func (r *Gojinn) Cleanup() error {
	if r.natsConn != nil {
		if err := r.natsConn.Drain(); err != nil {
			r.logger.Warn("NATS Drain error", zap.Error(err))
		}
		r.natsConn.Close()
	}

	if r.natsServer != nil {
		r.natsServer.Shutdown()
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

type wsContextKey struct{}

type HttpContext struct {
	W      http.ResponseWriter
	R      *http.Request
	WSConn *websocket.Conn
}
