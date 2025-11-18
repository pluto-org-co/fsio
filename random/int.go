// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
