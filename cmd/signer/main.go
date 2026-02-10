package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pauloappbr/gojinn/pkg/sovereign"
)

func main() {
	action := flag.String("action", "", "gen-keys | sign")
	name := flag.String("name", "gojinn", "Key name (for gen-keys)")
	keyFile := flag.String("key", "", "Path to the private key (for sign)")
	wasmFile := flag.String("file", "", "WASM file to sign")
	flag.Parse()

	switch *action {
	case "gen-keys":
		err := sovereign.GenerateKeys(*name)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Keys generated: %s.pub and %s.priv\n", *name, *name)
		pubBytes, _ := os.ReadFile(*name + ".pub")
		fmt.Printf("PUBLIC KEY:\n%s\n", string(pubBytes))

	case "sign":
		if *keyFile == "" || *wasmFile == "" {
			panic("You must provide --key and --file")
		}

		privHex, err := os.ReadFile(*keyFile)
		if err != nil {
			panic(fmt.Errorf("failed to read private key: %w", err))
		}
		privKeyBytes, err := sovereign.ParsePrivateKey(string(privHex))
		if err != nil {
			panic(err)
		}

		wasmBytes, err := os.ReadFile(*wasmFile)
		if err != nil {
			panic(fmt.Errorf("WASM FILE NOT FOUND or unreadable: %w", err))
		}

		if len(wasmBytes) == 0 {
			panic("The WASM file is empty! Check the build.")
		}

		signedBytes, err := sovereign.SignWasm(wasmBytes, privKeyBytes)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(*wasmFile, signedBytes, 0644)
		if err != nil {
			panic(err)
		}
		fmt.Printf("File successfully signed: %s (Size: %d bytes)\n", *wasmFile, len(signedBytes))

	default:
		fmt.Println("Usage: --action=gen-keys or --action=sign")
	}
}
