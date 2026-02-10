//go:build !wasip1 && !wasm

package sdk

import "errors"

type DBHandlerStub struct{}

func (d DBHandlerStub) Query(query string) ([]map[string]interface{}, error) {
	return nil, errors.New("cannot run sdk.DB on host machine (wasm only)")
}

var DB = DBHandlerStub{}

type KVStoreStub struct{}

func (k KVStoreStub) Set(key, value string)         {}
func (k KVStoreStub) Get(key string) (string, bool) { return "", false }

var KV = KVStoreStub{}
