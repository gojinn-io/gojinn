package main

import (
	"github.com/gojinn-io/gojinn/sdk"
)

func main() {
	// 1. Testa a sua nova Host Function de Bypass Seguro (Síncrono)
	responseStr := sdk.HttpGet("https://jsonplaceholder.typicode.com/todos/1")

	// 2. Retorna a resposta Síncrona imediatamente para o Caddy/HTTP
	sdk.SendJSON(map[string]interface{}{
		"status":        "success",
		"message":       "API Gateway mode is working! 🧞",
		"external_data": responseStr,
	})
}
