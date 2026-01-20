package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ResponsePayload defines the strict JSON structure Caddy expects to receive.
type ResponsePayload struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

// RequestPayload defines the structure sent by Caddy to the WASM module.
type RequestPayload struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	TraceID string              `json:"trace_id"`
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run contains the core logic, allowing dependency injection for testing
func run(in io.Reader, out io.Writer) error {
	// 1. Read input
	inputData, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	// 2. Prepare response
	response := ResponsePayload{
		Status: 200,
		Headers: map[string][]string{
			"Content-Type": {"text/plain"},
			"X-Gojinn":     {"Phase2"},
		},
		Body: fmt.Sprintf("👋 Hello from Gojinn! Received %d bytes.", len(inputData)),
	}

	// 3. Serialize and write
	return json.NewEncoder(out).Encode(response)
}
