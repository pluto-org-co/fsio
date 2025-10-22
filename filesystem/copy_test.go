package filesystem_test

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

func Test_Copy(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		assertions := assert.New(t)

		files := testsuite.GenerateLocations(100)

		randomSrc := randomfs.New(files, 32*1024*1024)

		tempDir, err := os.MkdirTemp("", "*")
		if !assertions.Nil(err, "failed create temporary directory") {
			return
		}
		defer os.RemoveAll(tempDir)
		src := directory.New(tempDir, 0o777, 0o777)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		err = filesystem.Copy(ctx, src, randomSrc)
		if !assertions.Nil(err, "failed to copy files") {
			return
		}

		t.Run("Copy", func(t *testing.T) {
			assertions := assert.New(t)

			tempDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(tempDir)

			dst := directory.New(tempDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()
			err = filesystem.Copy(ctx, dst, src)
			if !assertions.Nil(err, "failed to copy files") {
				return
			}
		})
		t.Run("CopyWorkers", func(t *testing.T) {
			assertions := assert.New(t)

			tempDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(tempDir)

			dst := directory.New(tempDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()
			err = filesystem.CopyWorkers(100, ctx, dst, src)
			if !assertions.Nil(err, "failed to copy files") {
				return
			}
		})
	})
}
