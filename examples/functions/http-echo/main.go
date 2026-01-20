package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// RequestPayload defines the structure sent by Caddy/Gojinn.
type RequestPayload struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
	TraceID string              `json:"trace_id"` // Added for Phase 2 observability
}

// ResponsePayload defines the strict JSON structure Caddy expects back.
type ResponsePayload struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func main() {
	// Execute core logic connecting Stdin to Stdout
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

// run isolates logic for testing
func run(in io.Reader, out io.Writer) error {
	var req RequestPayload

	// Decode input
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return sendError(out, 400, "Invalid JSON input: "+err.Error())
	}

	// Logic: Simple Authorization Check
	auth := req.Headers["Authorization"]
	if len(auth) == 0 || auth[0] != "secret" {
		return sendError(out, 401, "Unauthorized: Missing or wrong secret")
	}

	// Logic: Success Response
	resp := ResponsePayload{
		Status: 200,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Gojinn":     {"Phase-2"},
		},
		// Echo back details including the TraceID
		Body: fmt.Sprintf(`{"message": "Hello from Wasm!", "your_method": "%s", "your_path": "%s", "trace_id": "%s"}`,
			req.Method, req.URI, req.TraceID),
	}

	return json.NewEncoder(out).Encode(resp)
}

// sendError helper writes a structured error response
func sendError(out io.Writer, code int, msg string) error {
	resp := ResponsePayload{
		Status: code,
		Body:   fmt.Sprintf(`{"error": "%s"}`, msg),
	}
	return json.NewEncoder(out).Encode(resp)
}
