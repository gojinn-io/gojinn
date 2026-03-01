package gojinn

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/getsentry/sentry-go"
	"github.com/pauloappbr/gojinn/pkg/sovereign"
)

func (g *Gojinn) loadWasmSecurely(path string) ([]byte, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	g.logger.Debug("LoadWasm Raw", zap.String("file", path), zap.Int("size", len(rawBytes)))

	if len(g.TrustedKeys) == 0 {
		if g.SecurityPolicy == "strict" {
			return nil, fmt.Errorf("security policy is strict but no trusted keys are defined")
		}

		cleanBytes := sovereign.StripSignature(rawBytes)
		return cleanBytes, nil
	}

	var trusted []ed25519.PublicKey
	for _, k := range g.TrustedKeys {
		pk, err := sovereign.ParsePublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted key config: %w", err)
		}
		trusted = append(trusted, pk)
	}

	cleanBytes, err := sovereign.VerifyWasm(rawBytes, trusted)
	if err != nil {
		if g.SecurityPolicy == "strict" {
			g.logger.Error("BLOCKING UNSIGNED MODULE", zap.String("file", path), zap.Error(err))
			return nil, fmt.Errorf("module signature verification failed: %w", err)
		}

		g.logger.Warn("Security Audit Failed (Allowing run due to audit policy)",
			zap.String("file", path),
			zap.Error(err))

		return sovereign.StripSignature(rawBytes), nil
	}

	g.logger.Info("Module Signature Verified", zap.String("file", path), zap.Int("size_clean", len(cleanBytes)))
	return cleanBytes, nil
}

func (g *Gojinn) saveCrashDump(filename string, data []byte) {
	if g.CrashPath == "" {
		g.CrashPath = "./crashes"
	}
	if err := os.MkdirAll(g.CrashPath, 0755); err != nil {
		g.logger.Error("Failed to create crash directory", zap.Error(err))
		return
	}

	fullPath := filepath.Join(g.CrashPath, filename)

	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		g.logger.Error("Failed to write crash dump", zap.Error(err))
	} else {
		g.logger.Info("Crash Dump Saved (Time Travel Ready)", zap.String("file", fullPath))
	}

	if g.SentryDSN != "" {
		var snapshot CrashSnapshot
		if err := json.Unmarshal(data, &snapshot); err == nil {
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("wasm_file", snapshot.WasmFile)
				scope.SetExtra("input_payload", string(snapshot.Input))

				envBytes, _ := json.Marshal(snapshot.Env)
				scope.SetExtra("env_vars", string(envBytes))

				sentry.CaptureMessage(fmt.Sprintf("WASM Crash in %s: %s", snapshot.WasmFile, snapshot.Error))
			})
		}
	}
}
