package main

import (
	"fmt"
	"os"
	"unsafe"
)

//go:wasmimport gojinn host_ask_ai
func host_ask_ai(promptPtr, promptLen, outPtr, outMaxLen uint32) uint32

func main() {
	prompt := "Explain shortly: Why use WebAssembly in server-side?"

	// CORREÇÃO: Usar Fprintf no Stderr para não quebrar o JSON de resposta
	fmt.Fprintf(os.Stderr, "🤖 [WASM LOG] Perguntando à IA: '%s'...\n", prompt)

	ptrPrompt := uint32(uintptr(unsafe.Pointer(&[]byte(prompt)[0])))
	lenPrompt := uint32(len(prompt))

	responseBuf := make([]byte, 4096)
	ptrResp := uint32(uintptr(unsafe.Pointer(&responseBuf[0])))
	lenResp := uint32(len(responseBuf))

	bytesWritten := host_ask_ai(ptrPrompt, lenPrompt, ptrResp, lenResp)

	if bytesWritten == 0 {
		// Retorna um JSON de erro válido
		fmt.Println(`{"error": "No response from AI"}`)
		return
	}

	aiResponse := string(responseBuf[:int(bytesWritten)])

	// CORREÇÃO 2: Precisamos escapar as quebras de linha da IA para o JSON não quebrar
	// (Num cenário real usariamos encoding/json, mas aqui vai um truque rápido)
	// O fmt.Printf("%q") do Go faz o escape automático de strings!

	fmt.Printf(`{"role": "AI", "model": "llama3", "response": %q}`, aiResponse)
}
