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

package pathmod_test

import (
	"os"
	"testing"

	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/pathmod"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_PathMod(t *testing.T) {
	assertions := assert.New(t)

	tempDir, err := os.MkdirTemp("", "*")
	if !assertions.Nil(err, "failed to create temp") {
		return
	}
	defer os.RemoveAll(tempDir)
	localRoot := directory.New(tempDir, 0o777, 0o777)

	pmRoot := pathmod.New(localRoot, func(oldLocation []string) (newLocation []string) {
		newLocation = make([]string, 0, 1+len(oldLocation))
		newLocation = append(newLocation, "prepended")
		newLocation = append(newLocation, oldLocation...)
		return newLocation
	})

	t.Run("Testsuite", testsuite.TestFilesystem(t, pmRoot))
}
