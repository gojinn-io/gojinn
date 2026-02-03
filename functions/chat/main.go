package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unsafe"
)

//go:wasmimport gojinn host_ws_upgrade
func host_ws_upgrade() uint32

//go:wasmimport gojinn host_ws_read
func host_ws_read(outPtr, outMaxLen uint32) uint64

//go:wasmimport gojinn host_ws_write
func host_ws_write(msgPtr, msgLen uint32)

func main() {
	input, _ := io.ReadAll(os.Stdin)
	reqJSON := string(input)

	isWS := false
	if strings.Contains(reqJSON, "websocket") || strings.Contains(reqJSON, "Upgrade") {
		isWS = true
	}

	if isWS {
		success := host_ws_upgrade()
		if success == 1 {
			msgBuf := make([]byte, 4096)
			ptrMsg := uint32(uintptr(unsafe.Pointer(&msgBuf[0])))

			for {
				n := host_ws_read(ptrMsg, 4096)
				if n == 0 {
					break
				}

				msg := string(msgBuf[:n])
				reply := fmt.Sprintf("🤖 Gojinn (Go Standard) says: %s", msg)

				ptrReply := uint32(uintptr(unsafe.Pointer(unsafe.StringData(reply))))
				host_ws_write(ptrReply, uint32(len(reply)))
			}
			return
		}
	}

	fmt.Printf(`{"status": 200, "headers": {"Content-Type": ["text/plain"]}, "body": "Use a WebSocket client!"}`)
}
