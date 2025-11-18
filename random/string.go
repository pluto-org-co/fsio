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
