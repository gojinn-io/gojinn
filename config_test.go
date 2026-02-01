package gojinn

import (
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/stretchr/testify/assert"
)

func TestParseCaddyfile(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedPath string
		expectedArgs []string
		expectedEnv  map[string]string
		expectedPool int
		expectedMem  string
		expectedTime time.Duration
		shouldErr    bool
	}{
		{
			name:         "Basic Config",
			input:        `gojinn ./test.wasm`,
			expectedPath: "./test.wasm",
			shouldErr:    false,
		},
		{
			name: "Full Config",
			input: `gojinn ./app.wasm {
				args --foo --bar
				env API_KEY 12345
				env DEBUG true
				timeout 5s
				memory_limit 128MB
				pool_size 10
			}`,
			expectedPath: "./app.wasm",
			expectedArgs: []string{"--foo", "--bar"},
			expectedEnv:  map[string]string{"API_KEY": "12345", "DEBUG": "true"},
			expectedPool: 10,
			expectedMem:  "128MB",
			expectedTime: 5 * time.Second,
			shouldErr:    false,
		},
		{
			name: "Invalid Pool Size",
			input: `gojinn ./app.wasm {
				pool_size not_a_number
			}`,
			expectedPath: "./app.wasm",
			shouldErr:    false,
		},
		{
			name: "Invalid Timeout",
			input: `gojinn ./app.wasm {
				timeout forever
			}`,
			expectedPath: "./app.wasm",
			shouldErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := caddyfile.NewTestDispenser(tt.input)
			h := httpcaddyfile.Helper{Dispenser: d}

			handler, err := parseCaddyfile(h)

			if tt.shouldErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			g, ok := handler.(*Gojinn)
			assert.True(t, ok, "Handler should be of type *Gojinn")

			assert.Equal(t, tt.expectedPath, g.Path)

			if len(tt.expectedArgs) > 0 {
				assert.Equal(t, tt.expectedArgs, g.Args)
			}

			if len(tt.expectedEnv) > 0 {
				assert.Equal(t, tt.expectedEnv, g.Env)
			}

			if tt.expectedPool > 0 {
				assert.Equal(t, tt.expectedPool, g.PoolSize)
			}

			if tt.expectedMem != "" {
				assert.Equal(t, tt.expectedMem, g.MemoryLimit)
			}

			if tt.expectedTime > 0 {
				assert.Equal(t, caddy.Duration(tt.expectedTime), g.Timeout)
			}
		})
	}
}
