package gojinn

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/stretchr/testify/assert"
)

func compileTestWasm(t *testing.T, sourceCode, outName string) string {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	wasmPath := filepath.Join(tmpDir, outName)

	err := os.WriteFile(srcPath, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "build", "-o", wasmPath, srcPath)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile wasm: %v\nOutput: %s", err, out)
	}

	return wasmPath
}

func TestProvision_FullLifecycle(t *testing.T) {
	code := `package main; func main() {}`
	wasmPath := compileTestWasm(t, code, "lifecycle.wasm")

	r := &Gojinn{
		Path:        wasmPath,
		MemoryLimit: "10MB",
		Timeout:     caddy.Duration(5 * time.Second),
		PoolSize:    2,
		NatsPort:    4223,
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	err := r.Provision(ctx)

	assert.NoError(t, err)

	assert.NotNil(t, r.metrics, "Metrics struct should be initialized by metrics.go logic")

	assert.NotNil(t, r.natsConn, "NATS Connection should be active")
	assert.Equal(t, "CONNECTED", r.natsConn.Status().String())

	r.subsMu.Lock()
	numSubs := len(r.subs)
	r.subsMu.Unlock()
	assert.Equal(t, 2, numSubs, "Should have exactly 2 NATS subscriptions (workers)")

	err = r.Cleanup()
	assert.NoError(t, err)
}

func TestProvision_DefaultPoolSize(t *testing.T) {
	code := `package main; func main() {}`
	wasmPath := compileTestWasm(t, code, "autoscaling.wasm")

	r := &Gojinn{
		Path:     wasmPath,
		PoolSize: 0,
		NatsPort: 4224,
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})
	err := r.Provision(ctx)
	assert.NoError(t, err)

	r.subsMu.Lock()
	numSubs := len(r.subs)
	r.subsMu.Unlock()
	assert.Equal(t, 2, numSubs, "Default pool size should be 2")

	_ = r.Cleanup()
}

func TestProvision_FileNotFound(t *testing.T) {
	r := &Gojinn{
		Path: "./arquivo_fantasma.wasm",
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	err := r.Provision(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read wasm file")
}

func TestProvision_GracefulInvalidConfig(t *testing.T) {
	code := `package main; func main() {}`
	wasmPath := compileTestWasm(t, code, "graceful.wasm")

	r := &Gojinn{
		Path:        wasmPath,
		MemoryLimit: "BATATA",
		PoolSize:    1,
		NatsPort:    4225,
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	err := r.Provision(ctx)
	assert.NoError(t, err)

	r.subsMu.Lock()
	numSubs := len(r.subs)
	r.subsMu.Unlock()
	assert.Equal(t, 1, numSubs)

	_ = r.Cleanup()
}
