package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type CrashSnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Error     string            `json:"error"`
	Input     json.RawMessage   `json:"input"`
	Env       map[string]string `json:"env"`
	WasmFile  string            `json:"wasm_file"`
}

func init() {
	rootCmd.AddCommand(replayCmd)
}

var replayCmd = &cobra.Command{
	Use:   "replay [crash_file.json]",
	Short: "Time-Travel Debugging: Replay a crash dump locally",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		crashFile := args[0]
		fmt.Printf("Time-Travel Initiated: Loading %s...\n", crashFile)

		data, err := os.ReadFile(crashFile)
		if err != nil {
			fmt.Printf("Failed to read crash file: %v\n", err)
			os.Exit(1)
		}

		var snapshot CrashSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			fmt.Printf("Invalid crash dump format: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Original Crash Time: %s\n", snapshot.Timestamp.Format(time.RFC822))
		fmt.Printf("Original Error: %s\n", snapshot.Error)
		fmt.Printf("Module: %s\n", snapshot.WasmFile)

		wasmBytes, err := os.ReadFile(snapshot.WasmFile)
		if err != nil {
			base := filepath.Base(snapshot.WasmFile)
			fmt.Printf("Original path not found, trying local: %s\n", base)
			wasmBytes, err = os.ReadFile(base)
			if err != nil {
				localPath := "functions/" + base
				fmt.Printf("Trying: %s\n", localPath)
				wasmBytes, err = os.ReadFile(localPath)
				if err != nil {
					fmt.Printf("Failed to find WASM file. Ensure you are in the project root.\n")
					os.Exit(1)
				}
			}
		}

		ctx := context.Background()
		rConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
		r := wazero.NewRuntimeWithConfig(ctx, rConfig)
		defer r.Close(ctx)

		_, err = r.NewHostModuleBuilder("gojinn").
			NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, ptr, len uint32) {
			mem, _ := mod.Memory().Read(ptr, len)
			fmt.Printf("[REPLAY LOG] %s\n", string(mem))
		}).Export("host_log").
			NewFunctionBuilder().WithFunc(func() {}).Export("host_db_query").
			NewFunctionBuilder().WithFunc(func() {}).Export("host_kv_set").
			NewFunctionBuilder().WithFunc(func() uint64 { return 0 }).Export("host_kv_get").
			NewFunctionBuilder().WithFunc(func() uint32 { return 1 }).Export("host_mutex_lock").
			NewFunctionBuilder().WithFunc(func() uint32 { return 1 }).Export("host_mutex_unlock").
			NewFunctionBuilder().WithFunc(func() uint32 { return 0 }).Export("host_s3_put").
			NewFunctionBuilder().WithFunc(func() uint32 { return 0 }).Export("host_s3_get").
			NewFunctionBuilder().WithFunc(func() uint32 { return 0 }).Export("host_enqueue").
			NewFunctionBuilder().WithFunc(func() uint64 { return 0 }).Export("host_ask_ai").
			NewFunctionBuilder().WithFunc(func() uint32 { return 0 }).Export("host_ws_upgrade").
			NewFunctionBuilder().WithFunc(func() uint64 { return 0 }).Export("host_ws_read").
			NewFunctionBuilder().WithFunc(func() {}).Export("host_ws_write").
			Instantiate(ctx)

		if err != nil {
			fmt.Printf("Failed to build mock host: %v\n", err)
			os.Exit(1)
		}

		wasi_snapshot_preview1.MustInstantiate(ctx, r)

		stdinBuf := bytes.NewReader(snapshot.Input)

		modConfig := wazero.NewModuleConfig().
			WithStdin(stdinBuf).
			WithStdout(os.Stdout).
			WithStderr(os.Stderr).
			WithSysWalltime().
			WithArgs("gojinn-replay")

		for k, v := range snapshot.Env {
			modConfig = modConfig.WithEnv(k, v)
		}

		fmt.Println("Replaying Execution...")
		fmt.Println("---------------------------------------------------")

		code, err := r.CompileModule(ctx, wasmBytes)
		if err != nil {
			fmt.Printf("Compile error: %v\n", err)
			os.Exit(1)
		}

		_, err = r.InstantiateModule(ctx, code, modConfig)

		fmt.Println("\n---------------------------------------------------")
		if err != nil {
			fmt.Printf("REPLAY SUCCESS: The crash was reproduced!\n")
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Output finished without error (Did you fix the bug?)\n")
		}
	},
}
