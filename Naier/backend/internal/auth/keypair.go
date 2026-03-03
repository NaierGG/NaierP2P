package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

func Ed25519PublicKeyFromBase64(s string) ([]byte, error) {
	decoded, err := decodeBase64(s)
	if err != nil {
		return nil, err
	}

	if len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("ed25519 public key must be 32 bytes")
	}

	return decoded, nil
}

func VerifyEd25519Signature(publicKeyB64, message, signatureB64 string) bool {
	publicKey, err := decodeBase64(publicKeyB64)
	if err != nil {
		return false
	}

	signature, err := decodeBase64(signatureB64)
	if err != nil {
		return false
	}

	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(publicKey), []byte(message), signature)
}

func X25519PublicKeyFromBase64(s string) ([]byte, error) {
	decoded, err := decodeBase64(s)
	if err != nil {
		return nil, err
	}

	if len(decoded) != 32 {
		return nil, errors.New("x25519 public key must be 32 bytes")
	}

	return decoded, nil
}

func decodeBase64(s string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(s)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("invalid base64 value")
}
