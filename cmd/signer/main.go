package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pauloappbr/gojinn/pkg/sovereign"
)

func main() {
	action := flag.String("action", "", "gen-keys | sign")
	keyName := flag.String("name", "default-key", "Key name (for gen-keys)")
	privateKeyPath := flag.String("key", "", "Private key path (for sign)")
	wasmPath := flag.String("file", "", "WASM file to sign")
	flag.Parse()

	switch *action {
	case "gen-keys":
		err := sovereign.GenerateKeys(*keyName)
		if err != nil {
			panic(err)
		}

		fmt.Printf("✅ Keys generated: %s.pub and %s.priv\n", *keyName, *keyName)

		pubBytes, _ := os.ReadFile(*keyName + ".pub")
		fmt.Printf("📋 PUBLIC KEY (Copy to your config file):\n%s\n", string(pubBytes))

	case "sign":
		if *privateKeyPath == "" || *wasmPath == "" {
			panic("Both --key and --file are required")
		}

		privHex, _ := os.ReadFile(*privateKeyPath)
		privKeyBytes, _ := sovereign.ParsePrivateKey(string(privHex))

		wasmBytes, _ := os.ReadFile(*wasmPath)

		signedBytes, err := sovereign.SignWasm(wasmBytes, privKeyBytes)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(*wasmPath, signedBytes, 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("🔐 File successfully signed: %s\n", *wasmPath)

	default:
		fmt.Println("Usage: go run main.go --action=gen-keys --name=mykey")
		fmt.Println("Usage: go run main.go --action=sign --key=mykey.priv --file=app.wasm")
	}
}
