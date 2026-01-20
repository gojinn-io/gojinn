package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// RequestPayload (Standard Protocol)
type RequestPayload struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	TraceID string              `json:"trace_id"`
}

// ResponsePayload (Standard Protocol)
type ResponsePayload struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		// If we can't even read/write, print to stderr (logs)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	// 1. Consume Input (Protocol requirement)
	// Even if we don't use it, we must read to clear the buffer
	io.ReadAll(in)

	// 2. The Logic: Attempt to Allocate 100MB
	// If the host (Caddy) has a memory_limit < 100MB, the WASM VM will be killed HERE.
	targetSize := 100 * 1024 * 1024 // 100MB

	// We use a slice to force allocation on the Heap
	bloat := make([]byte, targetSize)

	// Touch memory to ensure the OS/Runtime actually commits the pages
	bloat[0] = 1
	bloat[targetSize-1] = 1

	// 3. Response (Only reached if the Leaker survived)
	resp := ResponsePayload{
		Status: 200,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Gojinn":     {"Leaker-Survivor"},
		},
		Body: fmt.Sprintf(`{"status": "success", "allocated_bytes": %d, "message": "I survived the memory limit!"}`, len(bloat)),
	}

	return json.NewEncoder(out).Encode(resp)
}
