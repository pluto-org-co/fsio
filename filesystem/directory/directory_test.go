package directory_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/randomfs"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_Directory(t *testing.T) {
	assertions := assert.New(t)

	files := testsuite.GenerateFilenames(100)

	randomRoot := randomfs.New(files, 32*1024*1024)

	tempDir, err := os.MkdirTemp("", "*")
	if !assertions.Nil(err, "failed to create temp") {
		return
	}
	defer os.RemoveAll(tempDir)
	localRoot := directory.New(tempDir, 0o777, 0o777)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	err = filesystem.Copy(ctx, localRoot, randomRoot)
	if !assertions.Nil(err, "failed to copy fs") {
		return
	}

	t.Run("Testsuite", testsuite.TestFilesystem(t, localRoot))

}
