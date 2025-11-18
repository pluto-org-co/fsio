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

package testsuite

import (
	"github.com/pluto-org-co/fsio/random"
)

func GenerateFilename(nParts int) (location []string) {
	if nParts == 0 {
		nParts = 1
	}

	var parts = make([]string, 0, nParts)
	for range nParts {
		parts = append(parts, random.InsecureString(5))
	}
	return parts
}

func GenerateLocations(n int) (files [][]string) {
	files = make([][]string, 0, n)
	for range n {
		files = append(files, GenerateFilename(random.InsecureInt(5)))
	}

	return files
}
