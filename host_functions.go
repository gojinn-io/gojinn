package gojinn

import (
	"context"
	"fmt"

	"github.com/coder/websocket"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

func (r *Gojinn) buildHostModule(ctx context.Context, engine wazero.Runtime) error {
	_, err := engine.NewHostModuleBuilder("gojinn").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			level := uint32(stack[0])
			//nolint:gosec
			ptr := uint32(stack[1])
			//nolint:gosec
			size := uint32(stack[2])
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
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			queryPtr := uint32(stack[0])
			//nolint:gosec
			queryLen := uint32(stack[1])
			//nolint:gosec
			outPtr := uint32(stack[2])
			//nolint:gosec
			outMaxLen := uint32(stack[3])

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

			//nolint:gosec
			bytesToWrite := uint32(len(jsonBytes))

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
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			keyPtr := uint32(stack[0])
			//nolint:gosec
			keyLen := uint32(stack[1])
			//nolint:gosec
			valPtr := uint32(stack[2])
			//nolint:gosec
			valLen := uint32(stack[3])

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

			if r.kv == nil {
				r.logger.Error("KV Store not ready yet")
				return
			}

			_, err := r.kv.PutString(key, val)
			if err != nil {
				r.logger.Error("KV Put Failed", zap.String("key", key), zap.Error(err))
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_kv_set").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			keyPtr := uint32(stack[0])
			//nolint:gosec
			keyLen := uint32(stack[1])
			//nolint:gosec
			outPtr := uint32(stack[2])
			//nolint:gosec
			outMaxLen := uint32(stack[3])

			kBytes, ok := mod.Memory().Read(keyPtr, keyLen)
			if !ok {
				stack[0] = 0
				return
			}
			key := string(kBytes)

			if r.kv == nil {
				stack[0] = 0xFFFFFFFFFFFFFFFF
				return
			}

			entry, err := r.kv.Get(key)
			if err != nil {
				stack[0] = 0xFFFFFFFFFFFFFFFF
				return
			}

			valBytes := entry.Value()
			//nolint:gosec
			bytesToWrite := uint32(len(valBytes))

			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
			}

			if !mod.Memory().Write(outPtr, valBytes[:bytesToWrite]) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("host_kv_get").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			keyPtr := uint32(stack[0])
			//nolint:gosec
			keyLen := uint32(stack[1])
			//nolint:gosec
			bodyPtr := uint32(stack[2])
			//nolint:gosec
			bodyLen := uint32(stack[3])
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
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			keyPtr := uint32(stack[0])
			//nolint:gosec
			keyLen := uint32(stack[1])
			//nolint:gosec
			outPtr := uint32(stack[2])
			//nolint:gosec
			outMaxLen := uint32(stack[3])

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

			//nolint:gosec
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
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			filePtr := uint32(stack[0])
			//nolint:gosec
			fileLen := uint32(stack[1])
			//nolint:gosec
			payloadPtr := uint32(stack[2])
			//nolint:gosec
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
				r.runAsyncJob(context.Background(), wasmFile, payload)
			}()

			r.logger.Info("Job enqueued in background", zap.String("file", wasmFile))
			stack[0] = 0
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_enqueue").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			promptPtr := uint32(stack[0])
			//nolint:gosec
			promptLen := uint32(stack[1])
			//nolint:gosec
			outPtr := uint32(stack[2])
			//nolint:gosec
			outMaxLen := uint32(stack[3])

			pBytes, ok := mod.Memory().Read(promptPtr, promptLen)
			if !ok {
				stack[0] = 0
				return
			}
			prompt := string(pBytes)

			aiResponse, err := r.askAI(prompt)
			if err != nil {
				aiResponse = fmt.Sprintf(`{"error": "%s"}`, err.Error())
				r.logger.Error("AI Host Function Failed", zap.Error(err))
			}

			respBytes := []byte(aiResponse)
			//nolint:gosec
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
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("host_ask_ai").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			val := ctx.Value(wsContextKey{})
			if val == nil {
				r.logger.Error("WS Upgrade called without HTTP context")
				stack[0] = 0
				return
			}

			httpCtx := val.(*HttpContext)

			c, err := websocket.Accept(httpCtx.W, httpCtx.R, &websocket.AcceptOptions{
				InsecureSkipVerify: true,
			})
			if err != nil {
				r.logger.Error("Failed to accept websocket", zap.Error(err))
				stack[0] = 0
				return
			}

			httpCtx.WSConn = c
			stack[0] = 1
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export("host_ws_upgrade").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			outPtr := uint32(stack[0])
			//nolint:gosec
			outMaxLen := uint32(stack[1])

			val := ctx.Value(wsContextKey{})
			if val == nil {
				stack[0] = 0
				return
			}
			httpCtx := val.(*HttpContext)
			if httpCtx.WSConn == nil {
				stack[0] = 0
				return
			}

			_, msgBytes, err := httpCtx.WSConn.Read(ctx)
			if err != nil {
				stack[0] = 0
				return
			}

			//nolint:gosec
			bytesToWrite := uint32(len(msgBytes))
			if bytesToWrite > outMaxLen {
				bytesToWrite = outMaxLen
			}

			if !mod.Memory().Write(outPtr, msgBytes[:bytesToWrite]) {
				stack[0] = 0
				return
			}

			stack[0] = uint64(bytesToWrite)
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("host_ws_read").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			//nolint:gosec
			msgPtr := uint32(stack[0])
			//nolint:gosec
			msgLen := uint32(stack[1])

			val := ctx.Value(wsContextKey{})
			if val == nil {
				return
			}
			httpCtx := val.(*HttpContext)
			if httpCtx.WSConn == nil {
				return
			}

			msgBytes, ok := mod.Memory().Read(msgPtr, msgLen)
			if !ok {
				return
			}

			err := httpCtx.WSConn.Write(ctx, websocket.MessageText, msgBytes)
			if err != nil {
				r.logger.Error("WS Write failed", zap.Error(err))
			}
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_ws_write").
		Instantiate(ctx)

	return err
}
