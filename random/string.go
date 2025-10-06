package random

import (
	"crypto/rand"
	"encoding/hex"
)

func String(length int) (s string) {
	raw := make([]byte, length)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}
