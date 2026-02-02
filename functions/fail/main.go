package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("🧨 [Chaos Worker] Starting processing...")

	// Simulates some processing time before failing
	time.Sleep(100 * time.Millisecond)

	// Simulates a fatal error (panic or exit code 1)
	// Gojinn will detect exit code != 0 and treat it as an error
	fmt.Println("💥 BOOM! Simulated fatal error.")
	os.Exit(1)
}
