package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	// The payload sent by the trigger arrives via stdin
	payload, _ := io.ReadAll(os.Stdin)

	// Simulates a heavy workload
	fmt.Printf("🔥 [BACKGROUND JOB] Processing task: %s\n", string(payload))
}
