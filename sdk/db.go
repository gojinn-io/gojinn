package sdk

import (
	"encoding/json"
	"unsafe"
)

//go:wasmimport gojinn host_db_query
func host_db_query(queryPtr uint32, queryLen uint32, outPtr uint32, outMaxLen uint32) uint32

type DBHandler struct{}

var DB = DBHandler{}

func (d DBHandler) Query(query string) ([]map[string]interface{}, error) {
	ptr := unsafe.Pointer(unsafe.StringData(query))
	queryPtr := uint32(uintptr(ptr))
	queryLen := uint32(len(query))

	capacity := uint32(65536) // Query executa uma instrução SQL no Host e retorna um array de objetos dinâmicos.

	buffer := make([]byte, capacity)
	outPtr := uint32(uintptr(unsafe.Pointer(&buffer[0])))

	written := host_db_query(queryPtr, queryLen, outPtr, capacity)

	jsonBytes := buffer[:written]
	var result []map[string]interface{}

	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	if len(result) > 0 {
		if errMsg, ok := result[0]["error"]; ok {
			return nil, jsonError(errMsg.(string))
		}
	}

	return result, nil
}

type dbError struct {
	msg string
}

func (e dbError) Error() string  { return e.msg }
func jsonError(msg string) error { return dbError{msg} }
