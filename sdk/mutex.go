//go:build wasip1 || wasm

package sdk

import "unsafe"

//go:wasmimport
func host_mutex_lock(kPtr uint32, kLen uint32, ttlSeconds uint32) uint32

//go:wasmimport
func host_mutex_unlock(kPtr uint32, kLen uint32) uint32

type MutexService struct{}

var Mutex = MutexService{}

func (m MutexService) TryLock(key string, ttlSeconds uint32) bool {
	kPtr := uintptr(unsafe.Pointer(unsafe.StringData(key)))
	kLen := uint32(len(key))

	success := host_mutex_lock(uint32(kPtr), kLen, ttlSeconds)
	return success == 1
}

func (m MutexService) Unlock(key string) bool {
	kPtr := uintptr(unsafe.Pointer(unsafe.StringData(key)))
	kLen := uint32(len(key))

	success := host_mutex_unlock(uint32(kPtr), kLen)
	return success == 1
}
