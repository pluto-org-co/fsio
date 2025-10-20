package pathmod_test

import (
	"os"
	"path"
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

	pmRoot := pathmod.New(localRoot, func(oldNew string) (newPath string) {
		return path.Join("prepended", oldNew)
	})

	t.Run("Testsuite", testsuite.TestFilesystem(t, pmRoot))
}
