package gojinn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
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
}

func (*Gojinn) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.gojinn",
		New: func() caddy.Module { return &Gojinn{} },
	}
}

func (r *Gojinn) Provision(ctx caddy.Context) error {
	r.logger = ctx.Logger()

	if err := r.setupMetrics(ctx); err != nil {
		return err
	}

	if err := r.setupDB(); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}

	if len(r.CronJobs) > 0 {
		r.scheduler = cron.New(cron.WithSeconds())

		for _, job := range r.CronJobs {
			j := job
			_, err := r.scheduler.AddFunc(j.Schedule, func() {
				r.runBackgroundJob(j.WasmFile)
			})

			if err != nil {
				return fmt.Errorf("failed to schedule cron job '%s': %v", j.Schedule, err)
			}
			r.logger.Info("Cron job scheduled", zap.String("schedule", j.Schedule), zap.String("wasm", j.WasmFile))
		}
		r.scheduler.Start()
	}

	if r.MQTTBroker != "" {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(r.MQTTBroker)

		clientID := r.MQTTClientID
		if clientID == "" {
			clientID = fmt.Sprintf("gojinn-%d", time.Now().UnixNano())
		}
		opts.SetClientID(clientID)

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
					payload := string(msg.Payload())
					r.logger.Debug("MQTT Message Received", zap.String("topic", msg.Topic()))

					go r.runAsyncJob(s.WasmFile, payload)
				})

				if token.Wait() && token.Error() != nil {
					r.logger.Error("MQTT Subscribe Failed", zap.String("topic", s.Topic), zap.Error(token.Error()))
				} else {
					r.logger.Info("MQTT Subscribed", zap.String("topic", s.Topic), zap.String("wasm", s.WasmFile))
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

	if r.Path == "" {
		return fmt.Errorf("wasm file path is required")
	}

	if r.PoolSize <= 0 {
		r.PoolSize = 2
	}
	if r.Timeout == 0 {
		r.Timeout = caddy.Duration(60 * time.Second)
	}

	if r.Path != "" {
		r.enginePool = make(chan *EnginePair, r.PoolSize)

		wasmBytes, err := os.ReadFile(r.Path)
		if err != nil {
			return fmt.Errorf("failed to read wasm file: %w", err)
		}

		r.logger.Info("provisioning worker pool",
			zap.Int("workers", r.PoolSize),
			zap.String("path", r.Path),
			zap.String("strategy", "parallel_boot"))

		startBoot := time.Now()
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

		if len(r.enginePool) == 0 {
			return fmt.Errorf("failed to provision any workers")
		}

		r.logger.Info("worker pool ready", zap.Duration("boot_time", time.Since(startBoot)))
	}

	return nil
}

func (r *Gojinn) Cleanup() error {
	if r.mqttClient != nil && r.mqttClient.IsConnected() {
		r.mqttClient.Disconnect(250)
		r.logger.Info("MQTT Disconnected")
	}

	if r.scheduler != nil {
		r.scheduler.Stop()
		r.logger.Info("Cron scheduler stopped")
	}

	if r.db != nil {
		r.logger.Info("closing database connection pool")
		r.db.Close()
	}

	if r.enginePool != nil {
		r.logger.Info("shutting down worker pool", zap.String("path", r.Path))
		close(r.enginePool)
		for pair := range r.enginePool {
			if pair != nil && pair.Runtime != nil {
				pair.Runtime.Close(context.Background())
			}
		}
	}
	return nil
}
