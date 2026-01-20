package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_Authorized(t *testing.T) {
	// 1. Valid Request with correct secret
	req := RequestPayload{
		Method:  "GET",
		URI:     "/protected",
		Headers: map[string][]string{"Authorization": {"secret"}},
		TraceID: "abc-123",
	}
	inputBytes, _ := json.Marshal(req)

	var stdout bytes.Buffer
	err := run(bytes.NewReader(inputBytes), &stdout)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 2. Validate Response
	var resp ResponsePayload
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	if !strings.Contains(resp.Body, "Hello from Wasm!") {
		t.Errorf("Body mismatch. Got: %s", resp.Body)
	}

	if !strings.Contains(resp.Body, "abc-123") {
		t.Errorf("TraceID not found in body. Got: %s", resp.Body)
	}
}

func TestRun_Unauthorized(t *testing.T) {
	// 1. Request with WRONG secret
	req := RequestPayload{
		Method:  "GET",
		URI:     "/protected",
		Headers: map[string][]string{"Authorization": {"wrong-password"}},
	}
	inputBytes, _ := json.Marshal(req)

	var stdout bytes.Buffer
	err := run(bytes.NewReader(inputBytes), &stdout)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var resp ResponsePayload
	json.Unmarshal(stdout.Bytes(), &resp)

	if resp.Status != 401 {
		t.Errorf("Expected status 401 for bad auth, got %d", resp.Status)
	}

	if !strings.Contains(resp.Body, "Unauthorized") {
		t.Errorf("Expected unauthorized message. Got: %s", resp.Body)
	}
}

func TestRun_InvalidJSON(t *testing.T) {
	// 1. Malformed JSON input
	input := []byte(`{ not valid json }`)

	var stdout bytes.Buffer
	err := run(bytes.NewReader(input), &stdout)

	// Note: run() handles the error by writing a 400 response, it returns nil (success execution)
	// unless writing to stdout fails.
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var resp ResponsePayload
	json.Unmarshal(stdout.Bytes(), &resp)

	if resp.Status != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.Status)
	}
}
