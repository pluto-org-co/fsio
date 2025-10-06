package filesystem_test

import (
	"compress/gzip"
	"context"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/pluto-org-co/fsio/random"
	"github.com/stretchr/testify/assert"
)

func Test_Gzip(t *testing.T) {
	assertions := assert.New(t)

	const TotalFiles = 1_000
	files := func() (files []string) {
		files = make([]string, TotalFiles)
		for index := range files {
			files[index] = random.String(10)
		}
		return files
	}()

	randomRoot := filesystem.NewRandom(files, 1024)

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

	gzipRoot := filesystem.NewGzip(gzip.BestCompression, localRoot)

	t.Run("Testsuite", testsuite.TestFilesystem(t, gzipRoot))
}
