package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_ValidProtocol(t *testing.T) {
	// 1. Setup: Simulate the JSON input Caddy would send
	inputPayload := RequestPayload{
		Method:  "POST",
		URI:     "/test",
		Body:    "Hello Gojinn",
		TraceID: "00-12345-00",
	}
	inputBytes, _ := json.Marshal(inputPayload)
	stdin := bytes.NewReader(inputBytes)

	// 2. Execution: Capture stdout
	var stdout bytes.Buffer
	err := run(stdin, &stdout)

	// 3. Assertions
	if err != nil {
		t.Fatalf("run() returned unexpected error: %v", err)
	}

	// Verify if output is valid JSON
	var response ResponsePayload
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nRaw Output: %s", err, stdout.String())
	}

	// Verify Contract Logic
	if response.Status != 200 {
		t.Errorf("Expected status 200, got %d", response.Status)
	}

	// Verify Headers
	if val, ok := response.Headers["X-Gojinn"]; !ok || val[0] != "Phase2" {
		t.Errorf("Missing or incorrect X-Gojinn header. Got: %v", response.Headers)
	}

	// Verify Body content (Logic check)
	expectedPart := "Hello from Gojinn!"
	if !strings.Contains(response.Body, expectedPart) {
		t.Errorf("Body does not contain expected message.\nExpected to contain: %s\nGot: %s", expectedPart, response.Body)
	}
}
