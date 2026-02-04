package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Request struct {
	Method  string              `json:"method"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

func main() {
	input, _ := io.ReadAll(os.Stdin)

	var req Request
	if len(input) > 0 {
		_ = json.Unmarshal(input, &req)
	}

	responseMsg := fmt.Sprintf("Hello from Gojinn! You sent: %s", req.Body)
	if req.Body == "" {
		responseMsg = "Hello from Gojinn! (No body sent)"
	}

	fmt.Printf(`{"status": 200, "headers": {"Content-Type": ["text/plain"]}, "body": "%s"}`, responseMsg)
}
