package random

import (
	"crypto/rand"
	"encoding/hex"
)

func CryptoString(length int) (s string) {
	raw := make([]byte, length)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}

func InsecureString(length int) (s string) {
	const Options = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var parts = make([]byte, length)
	for index := range parts {
		parts[index] = Options[insecureSrc.IntN(len(Options))]
	}

	return string(parts)
}
