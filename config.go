package gojinn

import (
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

type CronJob struct {
	Schedule string `json:"schedule"`
	WasmFile string `json:"wasm_file"`
}

type MQTTSub struct {
	Topic    string `json:"topic"`
	WasmFile string `json:"wasm_file"`
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Gojinn
	m.Env = make(map[string]string)
	m.Mounts = make(map[string]string)
	m.CronJobs = []CronJob{}
	m.MQTTSubs = []MQTTSub{}
	m.AllowedHosts = []string{}
	m.APIKeys = []string{}
	m.CorsOrigins = []string{}
	m.TrustedKeys = []string{}
	m.NatsRoutes = []string{}
	m.ClusterPeers = []string{}
	m.TrustedNatsUsers = []string{}

	m.RateLimit = 0
	m.RateBurst = 0
	m.CrashPath = "./crashes"

	for h.Next() {
		args := h.RemainingArgs()
		if len(args) > 0 {
			m.Path = args[0]
		}

		for h.NextBlock(0) {
			switch h.Val() {
			case "wasm_file", "path":
				if h.NextArg() {
					m.Path = h.Val()
				}
			case "env":
				if h.NextArg() {
					key := h.Val()
					if h.NextArg() {
						m.Env[key] = h.Val()
					}
				}
			case "mount":
				if h.NextArg() {
					hostDir := h.Val()
					if h.NextArg() {
						guestDir := h.Val()
						m.Mounts[hostDir] = guestDir
					}
				}
			case "args":
				m.Args = h.RemainingArgs()
			case "timeout":
				if h.NextArg() {
					val, err := caddy.ParseDuration(h.Val())
					if err == nil {
						m.Timeout = caddy.Duration(val)
					}
				}
			case "memory_limit":
				if h.NextArg() {
					m.MemoryLimit = h.Val()
				}
			case "fuel_limit":
				if h.NextArg() {
					val, err := strconv.ParseUint(h.Val(), 10, 64)
					if err == nil {
						m.FuelLimit = val
					}
				}
			case "pool_size":
				if h.NextArg() {
					val, err := strconv.Atoi(h.Val())
					if err == nil {
						m.PoolSize = val
					}
				}
			case "debug_secret":
				if h.NextArg() {
					m.DebugSecret = h.Val()
				}
			case "db_driver":
				if h.NextArg() {
					m.DBDriver = h.Val()
				}
			case "db_dsn":
				if h.NextArg() {
					m.DBDSN = h.Val()
				}

			case "db_sync_url":
				if h.NextArg() {
					m.DBSyncURL = h.Val()
				}
			case "db_sync_token":
				if h.NextArg() {
					m.DBSyncToken = h.Val()
				}

			case "s3_endpoint":
				if h.NextArg() {
					m.S3Endpoint = h.Val()
				}
			case "s3_region":
				if h.NextArg() {
					m.S3Region = h.Val()
				}
			case "s3_bucket":
				if h.NextArg() {
					m.S3Bucket = h.Val()
				}
			case "s3_access_key":
				if h.NextArg() {
					m.S3AccessKey = h.Val()
				}
			case "s3_secret_key":
				if h.NextArg() {
					m.S3SecretKey = h.Val()
				}

			case "permissions":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					switch h.Val() {
					case "kv_read":
						m.Perms.KVRead = append(m.Perms.KVRead, h.RemainingArgs()...)
					case "kv_write":
						m.Perms.KVWrite = append(m.Perms.KVWrite, h.RemainingArgs()...)
					case "s3_read":
						m.Perms.S3Read = append(m.Perms.S3Read, h.RemainingArgs()...)
					case "s3_write":
						m.Perms.S3Write = append(m.Perms.S3Write, h.RemainingArgs()...)
					}
				}

			case "ai_tool":
				m.ExposeAsTool = true
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					switch h.Val() {
					case "name":
						if h.NextArg() {
							m.ToolMeta.Name = h.Val()
						}
					case "description":
						if h.NextArg() {
							m.ToolMeta.Description = h.Val()
						}
					case "schema":
						if h.NextArg() {
							m.ToolMeta.InputSchema = h.Val()
						}
					}
				}

			case "security":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					switch h.Val() {
					case "policy":
						if !h.NextArg() {
							return nil, h.Err("security policy expects 'strict' or 'audit'")
						}
						m.SecurityPolicy = h.Val()
					case "trusted_key":
						if !h.NextArg() {
							return nil, h.Err("trusted_key expects a hex public key string")
						}
						m.TrustedKeys = append(m.TrustedKeys, h.Val())
					}
				}

			case "nats_port":
				if h.NextArg() {
					val, err := strconv.Atoi(h.Val())
					if err == nil {
						m.NatsPort = val
					} else {
						return nil, h.Err("nats_port expects an integer")
					}
				}
			case "nats_routes":
				m.NatsRoutes = append(m.NatsRoutes, h.RemainingArgs()...)

			case "nats_user_seed":
				if !h.NextArg() {
					return nil, h.Err("nats_user_seed requires a seed string (starts with SU...)")
				}
				m.NatsUserSeed = h.Val()

			case "trusted_nats_users":
				m.TrustedNatsUsers = append(m.TrustedNatsUsers, h.RemainingArgs()...)

			case "cron":
				var job CronJob
				if !h.NextArg() {
					return nil, h.Err("cron expects a schedule string")
				}
				job.Schedule = h.Val()
				if !h.NextArg() {
					return nil, h.Err("cron expects a wasm file path")
				}
				job.WasmFile = h.Val()
				m.CronJobs = append(m.CronJobs, job)

			case "mqtt_broker":
				if h.NextArg() {
					m.MQTTBroker = h.Val()
				}
			case "mqtt_client_id":
				if h.NextArg() {
					m.MQTTClientID = h.Val()
				}
			case "mqtt_username":
				if h.NextArg() {
					m.MQTTUsername = h.Val()
				}
			case "mqtt_password":
				if h.NextArg() {
					m.MQTTPassword = h.Val()
				}
			case "mqtt_subscribe":
				var sub MQTTSub
				if !h.NextArg() {
					return nil, h.Err("mqtt_subscribe expects a topic")
				}
				sub.Topic = h.Val()
				if !h.NextArg() {
					return nil, h.Err("mqtt_subscribe expects a wasm file path")
				}
				sub.WasmFile = h.Val()
				m.MQTTSubs = append(m.MQTTSubs, sub)

			case "ai_provider":
				if h.NextArg() {
					m.AIProvider = h.Val()
				}
			case "ai_model":
				if h.NextArg() {
					m.AIModel = h.Val()
				}
			case "ai_endpoint":
				if h.NextArg() {
					m.AIEndpoint = h.Val()
				}
			case "ai_token":
				if h.NextArg() {
					m.AIToken = h.Val()
				}

			case "api_key":
				if h.NextArg() {
					m.APIKeys = append(m.APIKeys, h.Val())
				}
			case "allow_host":
				if h.NextArg() {
					m.AllowedHosts = append(m.AllowedHosts, h.Val())
				}
			case "cors_origin":
				if h.NextArg() {
					m.CorsOrigins = append(m.CorsOrigins, h.Val())
				}

			case "rate_limit":
				if h.NextArg() {
					val, err := strconv.ParseFloat(h.Val(), 64)
					if err == nil {
						m.RateLimit = val
					}
				}
				if h.NextArg() {
					val, err := strconv.Atoi(h.Val())
					if err == nil {
						m.RateBurst = val
					}
				}

			case "record_crashes":
				if h.NextArg() {
					val := h.Val()
					if val == "true" {
						m.RecordCrashes = true
					}
				}
			case "crash_path":
				if h.NextArg() {
					m.CrashPath = h.Val()
				}

			case "data_dir":
				if h.NextArg() {
					m.DataDir = h.Val()
				}
			case "cluster_name":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				m.ClusterName = h.Val()
			case "cluster_port":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				port, err := strconv.Atoi(h.Val())
				if err != nil {
					return nil, h.Errf("invalid cluster_port: %v", err)
				}
				m.ClusterPort = port
			case "cluster_peers":
				m.ClusterPeers = append(m.ClusterPeers, h.RemainingArgs()...)

			case "cluster_replicas":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				val, err := strconv.Atoi(h.Val())
				if err != nil {
					return nil, h.Errf("invalid cluster_replicas: %v", err)
				}
				m.ClusterReplicas = val

			case "consensus":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					var policy ConsensusPolicy
					policy.Namespace = h.Val()

					for nesting2 := h.Nesting(); h.NextBlock(nesting2); {
						switch h.Val() {
						case "mode":
							if h.NextArg() {
								policy.Mode = h.Val()
							}
						case "stale_reads":
							if h.NextArg() {
								policy.StaleReads = h.Val() == "true"
							}
						}
					}
					m.Consensus = append(m.Consensus, policy)
				}

			case "leaf_remotes":
				m.LeafRemotes = append(m.LeafRemotes, h.RemainingArgs()...)
			case "leaf_port":
				if h.NextArg() {
					val, err := strconv.Atoi(h.Val())
					if err == nil {
						m.LeafPort = val
					}
				}
			case "server_name":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				m.ServerName = h.Val()
			}
		}
	}
	return &m, nil
}
