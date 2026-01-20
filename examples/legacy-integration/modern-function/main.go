package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// --- 1. Gojinn Contract Structures ---

// Request defines the strict structure received from Caddy via Stdin.
type Request struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	TraceID string              `json:"trace_id"`
}

// Response defines the strict structure sent back to Caddy via Stdout.
type Response struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

// --- 2. Main Entrypoint ---

func main() {
	// Execute the core logic using standard I/O streams.
	// This pattern facilitates unit testing (using bytes.Buffer) and error handling.
	if err := run(os.Stdin, os.Stdout); err != nil {
		// Log fatal errors to Stderr so Caddy can capture them in operational logs.
		// Never print plain text to Stdout, as it breaks the JSON contract.
		fmt.Fprintf(os.Stderr, "Fatal Wasm Error: %v\n", err)
		os.Exit(1)
	}
}

// --- 3. Business Logic ---

func run(in io.Reader, out io.Writer) error {
	// A. Parse Input
	var req Request
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		// If input is invalid, we should return a 400 Bad Request
		return sendError(out, 400, "Invalid JSON input provided to WASM")
	}

	// B. Logic: "Modern" Implementation
	// In a real scenario, this would be the optimized Go logic replacing the legacy Python/Java code.
	// Notice we use the TraceID to allow correlation between the legacy logs and modern logs.

	// Logs go to Stderr (Operational Observability)
	fmt.Fprintf(os.Stderr, "[%s] Processing high-speed calculation request...\n", req.TraceID)

	responsePayload := fmt.Sprintf(
		`{"message": "⚡ Gojinn Speed: Calculation done in microseconds!", "legacy_replaced": true, "trace": "%s"}`,
		req.TraceID,
	)

	// C. Send Response
	resp := Response{
		Status: 200,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Migration":  {"Strangler-Fig-Success"},
			"X-Powered-By": {"Gojinn/Wasm"},
		},
		Body: responsePayload,
	}

	return json.NewEncoder(out).Encode(resp)
}

// sendError helps to return a compliant JSON error response
func sendError(out io.Writer, code int, msg string) error {
	resp := Response{
		Status: code,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: fmt.Sprintf(`{"error": "%s"}`, msg),
	}
	return json.NewEncoder(out).Encode(resp)
}
