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
