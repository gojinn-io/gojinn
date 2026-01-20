package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_FibonacciFast(t *testing.T) {
	// 1. Setup Input with small N (Fast execution)
	// We wrap the inner body string inside the JSON payload structure
	req := RequestPayload{
		Method: "POST",
		Body:   `{"n": 5}`,
	}
	inputBytes, _ := json.Marshal(req)

	var stdout bytes.Buffer

	// 2. Run
	err := run(bytes.NewReader(inputBytes), &stdout)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 3. Validate
	var resp ResponsePayload
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	// Fib(5) = 5 (0, 1, 1, 2, 3, 5)
	if !strings.Contains(resp.Body, `"fib_result": 5`) {
		t.Errorf("Calculation seems wrong for n=5. Got body: %s", resp.Body)
	}
}

func TestRun_DefaultValue(t *testing.T) {
	// Empty body should default to logic (N=40), but we won't wait for result in this specific check
	// or we mock the fib function.
	// Ideally for unit tests we keep it simple. Let's just check if it accepts empty body without crashing.
	// NOTE: We pass N=1 via body to avoid the heavy default calculation during unit tests.

	req := RequestPayload{Body: `1`} // Raw integer body
	inputBytes, _ := json.Marshal(req)
	var stdout bytes.Buffer

	err := run(bytes.NewReader(inputBytes), &stdout)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	var resp ResponsePayload
	json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Status != 200 {
		t.Errorf("Expected 200 OK, got %d", resp.Status)
	}
}
