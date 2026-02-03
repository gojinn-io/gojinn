package sovereign

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

func GenerateKeys(prefix string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	if err := os.WriteFile(prefix+".priv", []byte(hex.EncodeToString(priv)), 0600); err != nil {
		return err
	}

	if err := os.WriteFile(prefix+".pub", []byte(hex.EncodeToString(pub)), 0644); err != nil {
		return err
	}

	return nil
}

func ParsePublicKey(hexKey string) (ed25519.PublicKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	if len(bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"invalid public key size (expected %d, got %d)",
			ed25519.PublicKeySize,
			len(bytes),
		)
	}

	return ed25519.PublicKey(bytes), nil
}

func ParsePrivateKey(hexKey string) (ed25519.PrivateKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	if len(bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"invalid private key size (expected %d, got %d)",
			ed25519.PrivateKeySize,
			len(bytes),
		)
	}

	return ed25519.PrivateKey(bytes), nil
}
