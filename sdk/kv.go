package sdk

import (
	"unsafe"
)

//go:wasmimport gojinn host_kv_set
func host_kv_set(kPtr, kLen, vPtr, vLen uint32)

//go:wasmimport gojinn host_kv_get
func host_kv_get(kPtr, kLen, outPtr, outMaxLen uint32) uint32

type KVStore struct{}

var KV = KVStore{}

func (k KVStore) Set(key, value string) {
	kPtr := uintptr(unsafe.Pointer(unsafe.StringData(key)))
	kLen := uint32(len(key))
	vPtr := uintptr(unsafe.Pointer(unsafe.StringData(value)))
	vLen := uint32(len(value))

	host_kv_set(uint32(kPtr), kLen, uint32(vPtr), vLen)
}

func (k KVStore) Get(key string) (string, bool) {
	kPtr := uintptr(unsafe.Pointer(unsafe.StringData(key)))
	kLen := uint32(len(key))

	capacity := uint32(4096)
	buffer := make([]byte, capacity)
	outPtr := uintptr(unsafe.Pointer(&buffer[0]))

	retLen := host_kv_get(uint32(kPtr), kLen, uint32(outPtr), capacity)

	if retLen == 0xFFFFFFFF {
		return "", false
	}

	return string(buffer[:retLen]), true
}
