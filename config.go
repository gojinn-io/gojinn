package gojinn

import (
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Gojinn
	m.Env = make(map[string]string)
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
			}
		}
	}
	return &m, nil
}
