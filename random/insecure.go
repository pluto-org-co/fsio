package random

import (
	"math/rand/v2"
)

// INSECURE seed used for deterministic result in test.
var seed [32]byte

func init() {
	copy(seed[:], "INSECURE")
}

var InsecureReader = rand.NewChaCha8(seed)
var insecureSrc = rand.New(InsecureReader)
