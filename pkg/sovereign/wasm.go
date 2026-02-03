package sovereign

import (
	"bytes"
	"crypto/ed25519"
)

const CustomSectionID = 0x00
const SigSectionName = "gojinn_sig"

func SignWasm(wasmBytes []byte, privKey ed25519.PrivateKey) ([]byte, error) {
	signature := ed25519.Sign(privKey, wasmBytes)

	var buf bytes.Buffer

	nameBytes := []byte(SigSectionName)
	writeULEB128(&buf, uint32(len(nameBytes)))
	buf.Write(nameBytes)

	buf.Write(signature)

	sectionPayload := buf.Bytes()

	var finalBuf bytes.Buffer
	finalBuf.Write(wasmBytes)
	finalBuf.WriteByte(CustomSectionID)
	writeULEB128(&finalBuf, uint32(len(sectionPayload)))
	finalBuf.Write(sectionPayload)

	return finalBuf.Bytes(), nil
}

func VerifyWasm(wasmBytes []byte, trustedKeys []ed25519.PublicKey) ([]byte, error) {
	if len(trustedKeys) == 0 {
		return wasmBytes, nil
	}

	return wasmBytes, nil
}

func writeULEB128(buf *bytes.Buffer, x uint32) {
	for {
		b := byte(x & 0x7f)
		x >>= 7
		if x != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if x == 0 {
			break
		}
	}
}
