package main

import (
	"fmt"
	"unsafe"
)

//go:wasmimport gojinn host_enqueue
func host_enqueue(namePtr, nameLen, payloadPtr, payloadLen uint32) uint32

func main() {
	target := "./functions/fail.wasm"
	payload := `{"test": "chaos_engineering"}`

	ptrTarget := uint32(uintptr(unsafe.Pointer(&[]byte(target)[0])))
	lenTarget := uint32(len(target))

	ptrPayload := uint32(uintptr(unsafe.Pointer(&[]byte(payload)[0])))
	lenPayload := uint32(len(payload))

	res := host_enqueue(ptrTarget, lenTarget, ptrPayload, lenPayload)

	var jsonResponse string

	if res == 0 {
		jsonResponse = `{
			"status": 200,
			"headers": {"Content-Type": "application/json"},
			"body": "{\"status\": \"success\", \"message\": \"Chaos job triggered! Check the logs to see the retries.\"}"
		}`
	} else {
		jsonResponse = `{
			"status": 500,
			"headers": {"Content-Type": "application/json"},
			"body": "{\"status\": \"error\", \"message\": \"Failed to enqueue job\"}"
		}`
	}

	fmt.Print(jsonResponse)
}
