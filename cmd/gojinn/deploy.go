package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy [path_to_function]",
	Short: "Compile and hot-deploy a function",
	Long:  `Detects language, compiles to WASM, updates the worker pool via Hot Reload.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		funcDir := args[0]
		funcName := filepath.Base(funcDir)

		fmt.Printf("Deploying function: %s\n", funcName)

		wasmPath, err := compileAny(funcDir, funcName)
		if err != nil {
			fmt.Printf("Compilation failed: %v\n", err)
			os.Exit(1)
		}

		finalPath := filepath.Join("functions", funcName+".wasm")
		if err := copyFile(wasmPath, finalPath); err != nil {
			fmt.Printf("Install failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Artifact installed at: %s\n", finalPath)

		if err := triggerHotReload(); err != nil {
			fmt.Printf("Hot Reload failed (Server down?): %v\n", err)
			fmt.Println("Manual restart required: Ctrl+C -> gojinn run")
		} else {
			fmt.Println("Hot Reload Triggered Successfully!")
		}
	},
}

func triggerHotReload() error {
	url := "http://localhost:8080/_sys/patch"
	payload := []byte(`{"reload": true}`)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

func compileAny(dir, name string) (string, error) {
	if exists(filepath.Join(dir, "Cargo.toml")) {
		return compileRust(dir, name)
	}
	if exists(filepath.Join(dir, "go.mod")) || exists(filepath.Join(dir, "main.go")) {
		return compileGo(dir, name)
	}
	if exists(filepath.Join(dir, "package.json")) {
		return compileJS(dir, name)
	}
	return "", fmt.Errorf("unknown language")
}

func compileRust(dir, _ string) (string, error) {
	fmt.Println("Building Rust...")
	exec.Command("cargo", "clean").Run()
	cmd := exec.Command("cargo", "build", "--release", "--target", "wasm32-wasip1")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return findNewestWasm(filepath.Join(dir, "target", "wasm32-wasip1", "release"))
}

func compileGo(dir, name string) (string, error) {
	fmt.Println("Building Go...")
	cmd := exec.Command("tinygo", "build", "-o", name+".wasm", "-target", "wasi", ".")
	cmd.Dir = dir
	if _, err := exec.LookPath("tinygo"); err != nil {
		cmd = exec.Command("go", "build", "-o", name+".wasm", ".")
		cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".wasm"), nil
}

func compileJS(dir, _ string) (string, error) {
	fmt.Println("Bundling JS...")
	if exists(filepath.Join(dir, "package.json")) {
		exec.Command("npm", "install").Run()
	}
	cmd := exec.Command("javy", "compile", filepath.Join(dir, "index.js"), "-o", filepath.Join(dir, "index.wasm"))
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.wasm"), nil
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0644)
}
func findNewestWasm(dir string) (string, error) {
	entries, _ := os.ReadDir(dir)
	var newest string
	var time int64
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wasm") {
			i, _ := e.Info()
			if i.ModTime().UnixNano() > time {
				time = i.ModTime().UnixNano()
				newest = filepath.Join(dir, e.Name())
			}
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no wasm found")
	}
	return newest, nil
}
