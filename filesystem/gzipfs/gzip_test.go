package gzipfs_test

import (
	"compress/gzip"
	"os"
	"testing"

	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/gzipfs"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_Gzip(t *testing.T) {
	assertions := assert.New(t)

	tempDir, err := os.MkdirTemp("", "*")
	if !assertions.Nil(err, "failed to create temp") {
		return
	}
	defer os.RemoveAll(tempDir)
	localRoot := directory.New(tempDir, 0o777, 0o777)

	gzipRoot := gzipfs.New(gzip.BestCompression, localRoot)

	t.Run("Testsuite", testsuite.TestFilesystem(t, gzipRoot))
}
