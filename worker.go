package gojinn

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/zap"
)

type EnginePair struct {
	Runtime wazero.Runtime
	Code    wazero.CompiledModule
}

func (r *Gojinn) createWorker(wasmBytes []byte) (*EnginePair, error) {
	ctxWazero := context.Background()
	rConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)

	if r.MemoryLimit != "" {
		bytes, err := humanize.ParseBytes(r.MemoryLimit)
		if err == nil && bytes > 0 {
			const wasmPageSize = 65536
			pages := uint32(bytes / wasmPageSize) //nolint:gosec
			if bytes%wasmPageSize != 0 {
				pages++
			}
			rConfig = rConfig.WithMemoryLimitPages(pages)
		}
	}

	engine := wazero.NewRuntimeWithConfig(ctxWazero, rConfig)

	_, err := engine.NewHostModuleBuilder("gojinn").
		// 1. LOG
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			level := uint32(stack[0]) //nolint:gosec
			ptr := uint32(stack[1])   //nolint:gosec
			size := uint32(stack[2])  //nolint:gosec
			msgBytes, ok := mod.Memory().Read(ptr, size)
			if !ok {
				return
			}
			msg := string(msgBytes)
			if level == 3 {
				r.logger.Error(msg)
			} else {
				r.logger.Info(msg)
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_log").
		// 2. DB QUERY
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			queryPtr := uint32(stack[0])  //nolint:gosec
			queryLen := uint32(stack[1])  //nolint:gosec
			outPtr := uint32(stack[2])    //nolint:gosec
			outMaxLen := uint32(stack[3]) //nolint:gosec

			qBytes, ok := mod.Memory().Read(queryPtr, queryLen)
			if !ok {
				stack[0] = 0
				return
			}
			query := string(qBytes)

			jsonBytes, err := r.executeQueryToJSON(query)
			if err != nil {
				jsonBytes = []byte(fmt.Sprintf(`[{"error": "%s"}]`, err.Error()))
			}

			bytesToWrite := uint32(len(jsonBytes)) //nolint:gosec

			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
				jsonBytes = jsonBytes[:bytesToWrite]
			}

			if !mod.Memory().Write(outPtr, jsonBytes) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_db_query").
		// 3. KV SET
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			keyPtr := uint32(stack[0]) //nolint:gosec
			keyLen := uint32(stack[1]) //nolint:gosec
			valPtr := uint32(stack[2]) //nolint:gosec
			valLen := uint32(stack[3]) //nolint:gosec

			kBytes, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				return
			}
			key := string(kBytes)

			vBytes, ok := mod.Memory().Read(valPtr, valLen)
			if !ok {
				return
			}
			val := string(vBytes)

			r.kvStore.Store(key, val)

		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_kv_set").
		// 4. KV GET
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			keyPtr := uint32(stack[0])    //nolint:gosec
			keyLen := uint32(stack[1])    //nolint:gosec
			outPtr := uint32(stack[2])    //nolint:gosec
			outMaxLen := uint32(stack[3]) //nolint:gosec

			kBytes, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				stack[0] = 0
				return
			}
			key := string(kBytes)

			val, ok := r.kvStore.Load(key)
			if !ok {
				stack[0] = uint64(0xFFFFFFFF)
				return
			}

			valueStr := val.(string)
			valBytes := []byte(valueStr)
			bytesToWrite := uint32(len(valBytes)) //nolint:gosec

			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
			}

			if !mod.Memory().Write(outPtr, valBytes[:bytesToWrite]) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_kv_get").
		// 5. S3 PUT
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			keyPtr := uint32(stack[0])  //nolint:gosec
			keyLen := uint32(stack[1])  //nolint:gosec
			bodyPtr := uint32(stack[2]) //nolint:gosec
			bodyLen := uint32(stack[3]) //nolint:gosec
			kBytes, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				stack[0] = 1
				return
			}
			key := string(kBytes)

			bBytes, ok := mod.Memory().Read(bodyPtr, bodyLen)
			if !ok {
				stack[0] = 1
				return
			}

			err := r.s3Put(ctx, key, bBytes)
			if err != nil {
				r.logger.Error("s3 put failed", zap.Error(err))
				stack[0] = 1
			} else {
				stack[0] = 0
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_s3_put").
		// 6. S3 GET
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			keyPtr := uint32(stack[0])    //nolint:gosec
			keyLen := uint32(stack[1])    //nolint:gosec
			outPtr := uint32(stack[2])    //nolint:gosec
			outMaxLen := uint32(stack[3]) //nolint:gosec

			kBytes, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				stack[0] = 0
				return
			}
			key := string(kBytes)

			valBytes, err := r.s3Get(ctx, key)
			if err != nil {
				r.logger.Error("s3 get failed", zap.Error(err))
				stack[0] = 0
				return
			}

			bytesToWrite := uint32(len(valBytes))
			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
			}

			if !mod.Memory().Write(outPtr, valBytes[:bytesToWrite]) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_s3_get").
		// 7. ASYNC QUEUE (Fire & Forget)
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			filePtr := uint32(stack[0])
			fileLen := uint32(stack[1])
			payloadPtr := uint32(stack[2])
			payloadLen := uint32(stack[3])

			fBytes, ok := mod.Memory().Read(filePtr, fileLen)
			if !ok {
				stack[0] = 1
				return
			}
			wasmFile := string(fBytes)

			pBytes, ok := mod.Memory().Read(payloadPtr, payloadLen)
			if !ok {
				stack[0] = 1
				return
			}
			payload := string(pBytes)

			go func() {
				r.runAsyncJob(wasmFile, payload)
			}()

			r.logger.Info("Job enqueued in background", zap.String("file", wasmFile))
			stack[0] = 0
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_enqueue").
		// --- 8. AI ENGINE (Phase 10) ---
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// Assinatura: host_ask_ai(promptPtr, promptLen, outPtr, outMaxLen) -> bytesWritten
			promptPtr := uint32(stack[0])
			promptLen := uint32(stack[1])
			outPtr := uint32(stack[2])
			outMaxLen := uint32(stack[3])

			// 1. Ler Prompt
			pBytes, ok := mod.Memory().Read(promptPtr, promptLen)
			if !ok {
				stack[0] = 0
				return
			}
			prompt := string(pBytes)

			// 2. Chamar IA (OpenAI / Ollama)
			aiResponse, err := r.askAI(prompt)
			if err != nil {
				// Em caso de erro, retornamos o erro como JSON para o WASM saber o que houve
				aiResponse = fmt.Sprintf(`{"error": "%s"}`, err.Error())
				r.logger.Error("AI Host Function Failed", zap.Error(err))
			}

			// 3. Escrever Resposta
			respBytes := []byte(aiResponse)
			bytesToWrite := uint32(len(respBytes))

			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
				respBytes = respBytes[:bytesToWrite]
			}

			if !mod.Memory().Write(outPtr, respBytes) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_ask_ai").
		// -----------------------------
		Instantiate(ctxWazero)

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate host module: %w", err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctxWazero, engine)

	code, err := engine.CompileModule(ctxWazero, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile wasm binary: %w", err)
	}

	return &EnginePair{Runtime: engine, Code: code}, nil
}

// runBackgroundJob (Cron Wrapper)
func (r *Gojinn) runBackgroundJob(wasmFile string) {
	cronPayload := `{"event_type": "cron", "source": "gojinn_scheduler"}`
	r.runAsyncJob(wasmFile, cronPayload)
}

// runAsyncJob com Retry e DLQ (Phase 9.3)
func (r *Gojinn) runAsyncJob(wasmFile, payload string) {
	const maxRetries = 3
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = r.executeOneShot(wasmFile, payload)
		if err == nil {
			r.logger.Debug("Async job finished", zap.String("file", wasmFile), zap.Int("attempt", attempt))
			return
		}

		r.logger.Warn("Async job failed", zap.String("file", wasmFile), zap.Int("attempt", attempt), zap.Error(err))

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	r.moveToDLQ(wasmFile, payload, err)
}

func (r *Gojinn) executeOneShot(wasmFile, payload string) error {
	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		return fmt.Errorf("read file error: %w", err)
	}

	eng, err := r.createWorker(wasmBytes)
	if err != nil {
		return fmt.Errorf("create worker error: %w", err)
	}
	defer eng.Runtime.Close(context.Background())

	modConfig := wazero.NewModuleConfig().
		WithStdin(bytes.NewBufferString(payload)).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	mod, err := eng.Runtime.InstantiateModule(context.Background(), eng.Code, modConfig)
	if err != nil {
		return err
	}

	return mod.Close(context.Background())
}

func (r *Gojinn) moveToDLQ(wasmFile, payload string, err error) {
	r.logger.Error("💀 [DLQ] Job moved to Dead Letter Queue",
		zap.String("file", wasmFile),
		zap.String("payload", payload),
		zap.String("final_error", err.Error()),
	)
}
