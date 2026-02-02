package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	// The MQTT message comes through stdin
	payload, _ := io.ReadAll(os.Stdin)

	fmt.Printf("🌡️ [IoT SENSOR] Received data: %s\n", string(payload))

	// Here you could save it to the database:
	// gojinn.DBExec("INSERT INTO sensors (data) VALUES (?)", string(payload))
}
