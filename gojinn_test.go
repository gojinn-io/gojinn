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

// Helper to compile a simple WASM for testing.
// Requires 'go' to be in the PATH.
func compileTestWasm(t *testing.T, sourceCode, outName string) string {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	wasmPath := filepath.Join(tmpDir, outName)

	err := os.WriteFile(srcPath, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Compiles using go build
	cmd := exec.Command("go", "build", "-o", wasmPath, srcPath)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile wasm: %v\nOutput: %s", err, out)
	}

	return wasmPath
}

func TestProvision_ValidatesConfig(t *testing.T) {
	// Minimal Go code for a valid WASM binary
	code := `package main; func main() {}`
	wasmPath := compileTestWasm(t, code, "empty.wasm")

	r := &Gojinn{
		Path:        wasmPath,
		MemoryLimit: "10MB",
		Timeout:     caddy.Duration(5 * time.Second),
	}

	// Mock of Caddy Context
	// Note: Caddy's NewContext initializes a default metrics registry,
	// which is required for our Provision method to succeed.
	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	// Should pass without error
	err := r.Provision(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r.engine)
	assert.NotNil(t, r.metrics, "Metrics struct should be initialized")
}

func TestProvision_InvalidMemoryLimit(t *testing.T) {
	code := `package main; func main() {}`
	wasmPath := compileTestWasm(t, code, "empty.wasm")

	r := &Gojinn{
		Path:        wasmPath,
		MemoryLimit: "INVALID_VALUE",
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	// Should fail on provision due to humanize parsing error
	err := r.Provision(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid memory_limit")
}

func TestProvision_FileNotFound(t *testing.T) {
	r := &Gojinn{
		Path: "./file_that_does_not_exist.wasm",
	}

	ctx, _ := caddy.NewContext(caddy.Context{Context: context.Background()})

	err := r.Provision(ctx)
	assert.Error(t, err)
}
