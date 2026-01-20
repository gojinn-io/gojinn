package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_AllocatesMemory(t *testing.T) {
	// 1. Setup Input
	input := []byte(`{"method":"POST"}`)
	var stdout bytes.Buffer

	// 2. Run logic (This will try to allocate 100MB on your dev machine)
	// Assuming your dev machine has > 100MB free RAM, this should pass.
	err := run(bytes.NewReader(input), &stdout)

	if err != nil {
		t.Fatalf("Unexpected error (OOM on test machine?): %v", err)
	}

	// 3. Validate Protocol
	var resp ResponsePayload
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	expectedMsg := "I survived the memory limit!"
	if !strings.Contains(resp.Body, expectedMsg) {
		t.Errorf("Expected body to contain success message. Got: %s", resp.Body)
	}
}
