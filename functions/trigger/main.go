package main

import (
	"fmt"
	"unsafe"
)

//go:wasmimport gojinn host_enqueue
func host_enqueue(namePtr, nameLen, payloadPtr, payloadLen uint32) uint32

func main() {
	target := "./functions/worker.wasm"
	payload := `{"job_id": 123, "action": "resize_image", "status": "pending"}`

	ptrTarget := uint32(uintptr(unsafe.Pointer(&[]byte(target)[0])))
	lenTarget := uint32(len(target))

	ptrPayload := uint32(uintptr(unsafe.Pointer(&[]byte(payload)[0])))
	lenPayload := uint32(len(payload))

	res := host_enqueue(ptrTarget, lenTarget, ptrPayload, lenPayload)

	// --- HERE IS THE FIX ---
	// We need to build the JSON response for the Host (the envelope)
	// Status must be an INT (200, 500, etc.)
	// Body must be a STRING (the JSON the user will see)

	var jsonResponse string

	if res == 0 {
		// Success: HTTP 200
		// Note that we escape quotes in the body: \"status\": \"success\"
		jsonResponse = `{
			"status": 200,
			"headers": {"Content-Type": "application/json"},
			"body": "{\"status\": \"success\", \"message\": \"Job successfully enqueued\"}"
		}`
	} else {
		// Error: HTTP 500
		jsonResponse = `{
			"status": 500,
			"headers": {"Content-Type": "application/json"},
			"body": "{\"status\": \"error\", \"message\": \"Failed to enqueue job\"}"
		}`
	}

	// Print the envelope for Gojinn
	fmt.Print(jsonResponse)
}
