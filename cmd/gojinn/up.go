package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pauloappbr/gojinn/pkg/sovereign"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(upCmd)
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Build functions, Sign binaries, and start the Sovereign Cloud",
	Long:  `Automatically compiles ./functions/*.go to WASM, signs them if a private key is found, and starts the Caddy server.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Building & Signing Gojinn Functions...")

		entries, err := os.ReadDir("functions")
		if err != nil {
			fmt.Printf("Failed to read functions directory: %v\n", err)
			return
		}

		var wg sync.WaitGroup
		buildErrors := false

		for _, entry := range entries {
			if entry.IsDir() {
				wg.Add(1)
				go func(funcName string) {
					defer wg.Done()
					if err := buildAndSignFunction(funcName); err != nil {
						fmt.Printf("Failed %s: %v\n", funcName, err)
						buildErrors = true
					} else {
						fmt.Printf("Ready: %s.wasm\n", funcName)
					}
				}(entry.Name())
			}
		}

		wg.Wait()

		if buildErrors {
			fmt.Println("Build/Sign errors occurred. Aborting startup.")
			os.Exit(1)
		}

		fmt.Println("\nStarting Sovereign Cloud (Caddy)...")
		fmt.Println("---------------------------------------")

		srvCmd := exec.Command("go", "run", "cmd/caddy/main.go", "run", "--config", "Caddyfile.mesh")
		srvCmd.Stdout = os.Stdout
		srvCmd.Stderr = os.Stderr
		srvCmd.Env = os.Environ()

		if err := srvCmd.Run(); err != nil {
			fmt.Printf("Server crash: %v\n", err)
			os.Exit(1)
		}
	},
}

func buildAndSignFunction(name string) error {
	src := filepath.Join("functions", name, "main.go")
	out := filepath.Join("functions", name+".wasm")

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build error: %v\n%s", err, string(output))
	}

	keyPath := "paulo.priv"
	if _, err := os.Stat(keyPath); err == nil {
		if err := signBinary(out, keyPath); err != nil {
			return fmt.Errorf("signing error: %w", err)
		}
	}

	return nil
}

func signBinary(wasmFile, keyFile string) error {
	privKeyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}
	privKey, err := sovereign.ParsePrivateKey(string(privKeyBytes))
	if err != nil {
		return err
	}

	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		return err
	}

	signedBytes, err := sovereign.SignWasm(wasmBytes, privKey)
	if err != nil {
		return err
	}

	return os.WriteFile(wasmFile, signedBytes, 0644)
}
