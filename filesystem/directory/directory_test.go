package directory_test

import (
	"os"
	"testing"

	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_Directory(t *testing.T) {
	assertions := assert.New(t)

	tempDir, err := os.MkdirTemp("", "*")
	if !assertions.Nil(err, "failed to create temp") {
		return
	}
	defer os.RemoveAll(tempDir)
	localRoot := directory.New(tempDir, 0o777, 0o777)

	t.Run("Testsuite", testsuite.TestFilesystem(t, localRoot))

}
