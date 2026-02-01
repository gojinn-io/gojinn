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

	DBDriver string `json:"db_driver,omitempty"`
	DBDSN    string `json:"db_dsn,omitempty"`
	kvStore  sync.Map

	db      *sql.DB
	logger  *zap.Logger
	metrics *gojinnMetrics

	enginePool chan *EnginePair
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

	if r.Path == "" {
		return fmt.Errorf("wasm file path is required")
	}

	if r.PoolSize <= 0 {
		r.PoolSize = 2
	}
	if r.Timeout == 0 {
		r.Timeout = caddy.Duration(60 * time.Second)
	}

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
	return nil
}

func (r *Gojinn) Cleanup() error {
	if r.db != nil {
		r.logger.Info("closing database connection pool")
		r.db.Close()
	}

	if r.enginePool == nil {
		return nil
	}

	r.logger.Info("shutting down worker pool", zap.String("path", r.Path))

	close(r.enginePool)
	for pair := range r.enginePool {
		if pair != nil && pair.Runtime != nil {
			pair.Runtime.Close(context.Background())
		}
	}
	return nil
}
