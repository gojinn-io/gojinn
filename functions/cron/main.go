package main

import (
	"fmt"
	"time"
)

func main() {
	// Prints to stdout (which Gojinn captures and forwards to the Caddy logs)
	fmt.Printf("⏰ [CRON GO 1.25] Executed at: %s\n", time.Now().Format(time.RFC3339))
}
