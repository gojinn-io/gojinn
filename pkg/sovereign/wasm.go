package sovereign

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
)

var MagicFooter = []byte{0x47, 0x4A, 0x53, 0x49, 0x47}

const (
	FooterSize = 5
	SigSize    = ed25519.SignatureSize
)

func StripSignature(data []byte) []byte {
	if len(data) < FooterSize+SigSize {
		return data
	}

	footer := data[len(data)-FooterSize:]
	if !bytes.Equal(footer, MagicFooter) {
		return data
	}

	totalOverhead := FooterSize + SigSize
	contentEnd := len(data) - totalOverhead

	return data[:contentEnd]
}

func SignWasm(wasmBytes []byte, privKey ed25519.PrivateKey) ([]byte, error) {
	cleanBytes := StripSignature(wasmBytes)

	signature := ed25519.Sign(privKey, cleanBytes)

	var buf bytes.Buffer
	buf.Write(cleanBytes)
	buf.Write(signature)
	buf.Write(MagicFooter)

	return buf.Bytes(), nil
}

func VerifyWasm(signedBytes []byte, trustedKeys []ed25519.PublicKey) ([]byte, error) {
	totalLen := len(signedBytes)
	minSize := FooterSize + SigSize

	if totalLen < minSize {
		return nil, fmt.Errorf("file too small to contain a signature")
	}

	footer := signedBytes[totalLen-FooterSize:]
	if !bytes.Equal(footer, MagicFooter) {
		return nil, fmt.Errorf("file does not contain a valid signature footer (missing magic footer)")
	}

	wasmEnd := totalLen - minSize
	originalWasm := signedBytes[:wasmEnd]
	signature := signedBytes[wasmEnd : totalLen-FooterSize]

	verified := false
	for _, key := range trustedKeys {
		if ed25519.Verify(key, originalWasm, signature) {
			verified = true
			break
		}
	}

	if !verified {
		return nil, fmt.Errorf("invalid digital signature or untrusted key")
	}

	return originalWasm, nil
}
