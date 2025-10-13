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

		tempSrc, err := os.MkdirTemp("", "testing-*")
		if !assertions.Nil(err, "failed to create temporary src directory") {
			return
		}
		defer os.RemoveAll(tempSrc)
		var src filesystem.Filesystem = directory.New(tempSrc, 0o777, 0o777)
		t.Run("Populate Source FS", func(t *testing.T) {
			assertions := assert.New(t)

			randomSrc := randomfs.New(testsuite.GenerateFilenames(5*1024), 5*1024*1024)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()
			err = filesystem.Copy(ctx, src, randomSrc)
			if !assertions.Nil(err, "failed to copy files from random") {
				return
			}
		})

		t.Run("Populate Destination FS", func(t *testing.T) {
			t.Run("Single Threaded", func(t *testing.T) {
				assertions := assert.New(t)

				tempDst, err := os.MkdirTemp("", "testing-*")
				if !assertions.Nil(err, "failed to create temporary dst directory") {
					return
				}
				defer os.RemoveAll(tempDst)
				dst := directory.New(tempDst, 0o777, 0o777)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()
				err = filesystem.Copy(ctx, dst, src)
				if !assertions.Nil(err, "failed to copy files") {
					return
				}
			})

			t.Run("Multi Threaded", func(t *testing.T) {
				assertions := assert.New(t)

				tempDst, err := os.MkdirTemp("", "testing-*")
				if !assertions.Nil(err, "failed to create temporary dst directory") {
					return
				}
				defer os.RemoveAll(tempDst)
				dst := directory.New(tempDst, 0o777, 0o777)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()
				err = filesystem.CopyWorkers(20, ctx, dst, src)
				if !assertions.Nil(err, "failed to copy files") {
					return
				}
			})
		})
	})
}
