package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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

// Custom body for this function
type FibRequest struct {
	N int `json:"n"`
}

// fib calculates Fibonacci recursively (CPU intensive)
func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	var req RequestPayload
	// 1. Decode Envelope
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return sendError(out, 400, "Invalid JSON input")
	}

	// 2. Parse Logic Input (Try to find 'n' in the body)
	// Default to 40 (Heavy load) if not provided
	n := 40
	var logicInput FibRequest
	if req.Body != "" {
		// Try to parse body as JSON
		if err := json.Unmarshal([]byte(req.Body), &logicInput); err == nil && logicInput.N > 0 {
			n = logicInput.N
		} else {
			// Try to parse body as raw integer string
			if val, err := strconv.Atoi(req.Body); err == nil && val > 0 {
				n = val
			}
		}
	}

	// 3. Do the Heavy Work (CPU Bound)
	// If timeout is configured in Caddy, this might be killed here.
	result := fib(n)

	// 4. Response
	resp := ResponsePayload{
		Status: 200,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"X-Gojinn":     {"Survivor"},
		},
		Body: fmt.Sprintf(`{"status": "success", "input": %d, "fib_result": %d, "message": "CPU work finished"}`, n, result),
	}

	return json.NewEncoder(out).Encode(resp)
}

func sendError(out io.Writer, code int, msg string) error {
	resp := ResponsePayload{
		Status: code,
		Body:   fmt.Sprintf(`{"error": "%s"}`, msg),
	}
	return json.NewEncoder(out).Encode(resp)
}
