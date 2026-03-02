//go:build wasip1 || wasm

package sdk

import "unsafe"

//go:wasmimport gojinn host_http_get
func host_http_get(urlPtr, urlLen, outPtr, outMaxLen uint32) uint64

func HttpGet(url string) string {
	uPtr := uintptr(unsafe.Pointer(unsafe.StringData(url)))
	uLen := uint32(len(url))

	capacity := uint32(2 * 1024 * 1024)
	buffer := make([]byte, capacity)
	outPtr := uintptr(unsafe.Pointer(&buffer[0]))

	written := host_http_get(uint32(uPtr), uLen, uint32(outPtr), capacity)

	if written == 0 {
		return ""
	}

	return string(buffer[:written])
}
