package gojinn

import (
	"bytes"
	"context"
	"crypto/sha256" // NOVO: Para o hash do cache
	"database/sql"
	"encoding/hex" // NOVO: Para string do hash
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// Cron Jobs
	CronJobs  []CronJob  `json:"cron_jobs,omitempty"`
	scheduler *cron.Cron // interno

	// MQTT Config
	MQTTBroker   string      `json:"mqtt_broker,omitempty"`
	MQTTClientID string      `json:"mqtt_client_id,omitempty"`
	MQTTUsername string      `json:"mqtt_username,omitempty"`
	MQTTPassword string      `json:"mqtt_password,omitempty"`
	MQTTSubs     []MQTTSub   `json:"mqtt_subs,omitempty"`
	mqttClient   mqtt.Client // interno

	// AI Config
	AIProvider string `json:"ai_provider,omitempty"`
	AIModel    string `json:"ai_model,omitempty"`
	AIEndpoint string `json:"ai_endpoint,omitempty"`
	AIToken    string `json:"ai_token,omitempty"`

	// NOVO: Cache de IA
	aiCache sync.Map
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

	// 1. Cron
	if len(r.CronJobs) > 0 {
		r.scheduler = cron.New(cron.WithSeconds())
		for _, job := range r.CronJobs {
			j := job
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

	// 2. MQTT
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

	// Defaults & Worker Pool
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
		r.logger.Info("provisioning worker pool", zap.Int("workers", r.PoolSize))

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
	}
	if r.scheduler != nil {
		r.scheduler.Stop()
	}
	if r.db != nil {
		r.db.Close()
	}
	if r.enginePool != nil {
		close(r.enginePool)
		for pair := range r.enginePool {
			if pair != nil && pair.Runtime != nil {
				pair.Runtime.Close(context.Background())
			}
		}
	}
	return nil
}

// --- AI HELPER FUNCTIONS (Phase 10) ---

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

// askAI com Caching
func (g *Gojinn) askAI(prompt string) (string, error) {
	provider := g.AIProvider
	if provider == "" {
		provider = "openai"
	}

	model := g.AIModel
	if model == "" {
		if provider == "ollama" {
			model = "llama3"
		} else {
			model = "gpt-3.5-turbo"
		}
	}

	// 1. SMART CACHING CHECK
	// Gera um hash único para a combinação de Modelo + Prompt
	cacheKey := fmt.Sprintf("%s:%s", model, hashString(prompt))

	if cachedVal, ok := g.aiCache.Load(cacheKey); ok {
		g.logger.Debug("🧠 AI Cache Hit", zap.String("key", cacheKey))
		return cachedVal.(string), nil
	}

	// 2. Configura Endpoint
	endpoint := g.AIEndpoint
	if endpoint == "" {
		if provider == "ollama" {
			endpoint = "http://localhost:11434/v1/chat/completions"
		} else if provider == "gemini" {
			// Placeholder para Gemini (requer estrutura JSON diferente geralmente)
			endpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
		} else {
			endpoint = "https://api.openai.com/v1/chat/completions"
		}
	}

	// 3. Monta Request (Compatível com OpenAI/Ollama)
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

		// 4. SMART CACHING SAVE
		// Salva a resposta no cache para o futuro
		g.aiCache.Store(cacheKey, responseContent)
		g.logger.Debug("🧠 AI Cache Stored", zap.String("key", cacheKey))

		return responseContent, nil
	}

	return "", fmt.Errorf("AI returned no response")
}

// Helper para gerar Hash
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
