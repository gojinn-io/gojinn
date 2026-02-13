package gojinn

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"go.uber.org/zap"
)

var (
	activeServers   = make(map[int]*server.Server)
	activeServersMu sync.Mutex
)

func (g *Gojinn) startEmbeddedNATS() error {
	activeServersMu.Lock()
	defer activeServersMu.Unlock()

	if existing, ok := activeServers[g.NatsPort]; ok {
		g.natsServer = existing
		if g.logger != nil {
			g.logger.Info("NATS server already running for this port, reusing instance", zap.Int("port", g.NatsPort))
		}
		return g.connectLocalClient()
	}

	storeDir := filepath.Join(g.DataDir, "nats_store")

	fmt.Printf("\n--- STARTING NATS ---\n")
	fmt.Printf("Server Name: %s\n", g.ServerName)
	fmt.Printf("Cluster Name: %s\n", g.ClusterName)
	fmt.Printf("Store Dir: %s\n", storeDir)

	var routes []*url.URL
	for _, peer := range g.ClusterPeers {
		u, err := url.Parse(peer)
		if err != nil {
			g.logger.Warn("Invalid cluster peer URL", zap.String("url", peer), zap.Error(err))
			continue
		}
		routes = append(routes, u)
	}

	var leafRemotes []*server.RemoteLeafOpts
	for _, remoteUrl := range g.LeafRemotes {
		u, err := url.Parse(remoteUrl)
		if err != nil {
			g.logger.Warn("Invalid leaf remote URL", zap.String("url", remoteUrl), zap.Error(err))
			continue
		}

		remoteOpt := &server.RemoteLeafOpts{
			URLs: []*url.URL{u},
		}

		if g.NatsUserSeed != "" {
			remoteOpt.Nkey = g.NatsUserSeed
		}

		leafRemotes = append(leafRemotes, remoteOpt)
	}

	leafPort := g.LeafPort
	if leafPort == 0 {
		leafPort = 7422
	}

	var nkeyUsers []*server.NkeyUser
	if len(g.TrustedNatsUsers) > 0 {
		for _, pubKey := range g.TrustedNatsUsers {
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey: pubKey,
			})
		}
	}

	opts := &server.Options{
		ServerName:         g.ServerName,
		Port:               g.NatsPort,
		NoLog:              false,
		NoSigs:             true,
		JetStream:          true,
		StoreDir:           storeDir,
		JetStreamMaxStore:  1 * 1024 * 1024 * 1024,
		JetStreamMaxMemory: 64 * 1024 * 1024,

		Nkeys: nkeyUsers,

		Cluster: server.ClusterOpts{
			Name: g.ClusterName,
			Port: g.ClusterPort,
			Host: "0.0.0.0",
		},
		Routes: routes,

		LeafNode: server.LeafNodeOpts{
			Host:    "0.0.0.0",
			Port:    leafPort,
			Remotes: leafRemotes,
		},
	}

	if g.StoreCipherKey != "" {
		g.logger.Warn("🔒 Disk Encryption At-Rest is ENABLED. Do not lose the cipher key!")
		opts.JetStreamKey = g.StoreCipherKey
		opts.JetStreamCipher = server.AES
	}

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

	ns.ConfigureLogger()
	g.natsServer = ns

	go ns.Start()

	if !ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("nats server failed to start (check logs above)")
	}

	activeServers[g.NatsPort] = ns

	g.logger.Info("Embedded NATS JetStream Started",
		zap.String("url", ns.ClientURL()),
		zap.String("store_dir", storeDir),
		zap.Int("leaf_remotes", len(leafRemotes)),
	)

	return g.connectLocalClient()
}

func (g *Gojinn) connectLocalClient() error {
	if g.natsServer == nil {
		return fmt.Errorf("cannot connect local client: nats server is nil")
	}

	clientURL := g.natsServer.ClientURL()
	var connectOpts []nats.Option

	if g.NatsUserSeed != "" {
		kp, err := nkeys.FromSeed([]byte(g.NatsUserSeed))
		if err != nil {
			return fmt.Errorf("invalid nats_user_seed format: %w", err)
		}

		pub, err := kp.PublicKey()
		if err != nil {
			return fmt.Errorf("failed to get public key: %w", err)
		}

		connectOpts = append(connectOpts, nats.Nkey(pub, func(nonce []byte) ([]byte, error) {
			sig, err := kp.Sign(nonce)
			if err != nil {
				return nil, err
			}
			return sig, nil
		}))
	}

	nc, err := nats.Connect(clientURL, connectOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to local NATS: %w", err)
	}
	g.natsConn = nc

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("failed to init JetStream context: %w", err)
	}
	g.js = js

	go g.ensureJetStreamResources()

	return nil
}

func (g *Gojinn) ensureJetStreamResources() {
	streamName := "GOJINN_WORKER"
	kvBucket := "GOJINN_STATE"

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if g.js == nil {
			continue
		}

		_, err := g.js.StreamInfo(streamName)
		if err != nil {
			g.logger.Info("Attempting to initialize Durable Stream...", zap.String("stream", streamName), zap.Int("replicas", g.ClusterReplicas))
			_, err = g.js.AddStream(&nats.StreamConfig{
				Name:      streamName,
				Subjects:  []string{"gojinn.exec.>"},
				Storage:   nats.FileStorage,
				Retention: nats.WorkQueuePolicy,
				Replicas:  g.ClusterReplicas,
			})
			if err != nil {
				g.logger.Warn("Stream creation pending...", zap.Error(err))
				continue
			}
			g.logger.Info("Durable Stream Ready!", zap.String("stream", streamName))
		}

		if g.kv == nil {
			kv, err := g.js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket:      kvBucket,
				Description: "Gojinn Distributed State",
				Storage:     nats.FileStorage,
				History:     1,
				TTL:         0,
				Replicas:    g.ClusterReplicas,
			})

			if err != nil {
				g.logger.Warn("KV Bucket creation pending...", zap.Error(err))
				continue
			}

			g.kv = kv
			g.logger.Info("Distributed KV Store Ready!", zap.String("bucket", kvBucket))
		}

		go func() {
			monitorTicker := time.NewTicker(5 * time.Second)
			defer monitorTicker.Stop()

			for range monitorTicker.C {
				if g.js == nil {
					continue
				}

				info, err := g.js.StreamInfo(streamName)
				if err == nil && g.metrics != nil && g.metrics.queueDepth != nil {
					g.metrics.queueDepth.WithLabelValues(streamName).Set(float64(info.State.Msgs))
				}
			}
		}()

		return
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
