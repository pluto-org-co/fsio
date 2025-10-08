package filesystem_test

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_PathMod(t *testing.T) {
	assertions := assert.New(t)

	files := testsuite.GenerateFilenames(100)

	randomRoot := filesystem.NewRandom(files, 32*1024*1024)

	tempDir, err := os.MkdirTemp("", "*")
	if !assertions.Nil(err, "failed to create temp") {
		return
	}
	defer os.RemoveAll(tempDir)
	localRoot := filesystem.NewLocal(tempDir, 0o777, 0o777)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	err = filesystem.Copy(ctx, localRoot, randomRoot)
	if !assertions.Nil(err, "failed to copy fs") {
		return
	}

	pmRoot := filesystem.NewPathMod(localRoot, func(oldNew string) (newPath string) {
		return path.Join("prepended", oldNew)
	})

	t.Run("Testsuite", testsuite.TestFilesystem(t, pmRoot))
}
