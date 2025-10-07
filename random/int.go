package random

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/exp/constraints"
)

func CryptoInt[T constraints.Integer](max T) (n T) {
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if r != nil {
		n = T(r.Int64())
	}
	return n
}

func InsecureInt[T constraints.Integer](max T) (n T) {
	return T(insecureSrc.Int64N(int64(max)))
}
