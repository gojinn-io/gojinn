package gojinn

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func (g *Gojinn) startEmbeddedNATS() error {
	storeDir := filepath.Join(g.DataDir, "nats_store")

	// --- DEBUG: Imprime configuração no console ---
	fmt.Printf("\n--- STARTING NATS ---\n")
	fmt.Printf("Server Name: %s\n", g.ServerName)
	fmt.Printf("Cluster Name: %s\n", g.ClusterName)
	fmt.Printf("Cluster Port: %d\n", g.ClusterPort)
	fmt.Printf("Client Port: %d\n", g.NatsPort)
	fmt.Printf("Store Dir: %s\n", storeDir)
	// ---------------------------------------------

	var routes []*url.URL
	for _, peer := range g.ClusterPeers {
		u, err := url.Parse(peer)
		if err != nil {
			g.logger.Warn("Invalid cluster peer URL", zap.String("url", peer), zap.Error(err))
			continue
		}
		routes = append(routes, u)
	}
	opts := &server.Options{
		ServerName: g.ServerName,
		Port:       g.NatsPort,
		// NoLog false não adianta se não configurarmos o logger
		// Vamos deixar false, mas confiar no fmt.Printf acima para debug inicial
		NoLog:              false,
		NoSigs:             true,
		JetStream:          true,
		StoreDir:           storeDir,
		JetStreamMaxStore:  1 * 1024 * 1024 * 1024,
		JetStreamMaxMemory: 64 * 1024 * 1024,

		Cluster: server.ClusterOpts{
			Name: g.ClusterName,
			Port: g.ClusterPort,
			Host: "0.0.0.0",
		},
		Routes: routes,
	}

	// Fallback de segurança
	if opts.ServerName == "" {
		opts.ServerName = fmt.Sprintf("gojinn-node-%d", g.ClusterPort)
	}

	if len(g.NatsRoutes) > 0 {
		opts.Routes = server.RoutesFromStr(strings.Join(g.NatsRoutes, ","))
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create NATS server struct: %w", err)
	}

	// --- DEBUG: Configura Logger do NATS para stdout ---
	ns.ConfigureLogger()
	// --------------------------------------------------

	g.natsServer = ns

	// Inicia em goroutine
	go ns.Start()

	// Espera até estar pronto ou falhar
	if !ns.ReadyForConnections(10 * time.Second) {
		// Se falhar, tentamos pegar o último erro do logger (se possível) ou apenas retornamos
		return fmt.Errorf("nats server failed to start (check logs above)")
	}

	clientURL := ns.ClientURL()
	g.logger.Info("Embedded NATS JetStream Started",
		zap.String("url", clientURL),
		zap.String("store_dir", storeDir),
	)

	nc, err := nats.Connect(clientURL)
	if err != nil {
		return fmt.Errorf("failed to connect to local NATS: %w", err)
	}
	g.natsConn = nc

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("failed to init JetStream context: %w", err)
	}
	g.js = js

	// --- CORREÇÃO AQUI ---
	// Em vez de retornar erro se a stream falhar (o que mata o servidor no boot),
	// vamos iniciar uma goroutine que tenta criar a stream em background.
	// Isso permite que o servidor suba e os nós se conectem.
	go g.ensureStreamAsync()

	return nil
}

func (g *Gojinn) ensureStreamAsync() {
	streamName := "GOJINN_WORKER"
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Tenta obter info da stream
			_, err := g.js.StreamInfo(streamName)
			if err == nil {
				// Já existe, sucesso!
				return
			}

			// Tenta criar
			g.logger.Info("Attempting to initialize Durable Stream...", zap.String("stream", streamName))
			_, err = g.js.AddStream(&nats.StreamConfig{
				Name:      streamName,
				Subjects:  []string{"gojinn.exec.>"},
				Storage:   nats.FileStorage,
				Retention: nats.WorkQueuePolicy,
				// Opcional: Replicas: 2 (se quiser forçar replicação)
			})

			if err == nil {
				g.logger.Info("Durable Stream Created Successfully!", zap.String("stream", streamName))
				return
			}

			// Se falhar (ex: cluster not ready), apenas loga e tenta de novo no próximo tick
			g.logger.Warn("Stream creation pending (waiting for cluster quorum)...", zap.Error(err))
		}
	}
}

func (g *Gojinn) ReloadWorkers() error {
	g.logger.Info("Hot Reload Initiated: Recycling Workers...")

	g.subsMu.Lock()
	defer g.subsMu.Unlock()

	for _, sub := range g.subs {
		if err := sub.Drain(); err != nil {
			g.logger.Warn("Failed to drain worker sub", zap.Error(err))
		}
	}
	g.subs = nil

	wasmBytes, err := g.loadWasmSecurely(g.Path)
	if err != nil {
		return fmt.Errorf("failed to reload wasm file: %w", err)
	}

	topic := g.getFunctionTopic()

	for i := 0; i < g.PoolSize; i++ {
		sub, err := g.startWorkerSubscriber(i, topic, wasmBytes)
		if err != nil {
			return fmt.Errorf("failed to start new worker %d: %w", i, err)
		}
		g.subs = append(g.subs, sub)
	}

	g.logger.Info("Hot Reload Complete", zap.Int("new_workers", len(g.subs)))
	return nil
}

func (g *Gojinn) getFunctionTopic() string {
	return fmt.Sprintf("gojinn.exec.%s", hashString(g.Path))
}
