package filesystem_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/random"
	"github.com/stretchr/testify/assert"
)

func Test_Copy(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		assertions := assert.New(t)

		files := func() (files []string) {
			files = make([]string, 1_000)
			for index := range files {
				files[index] = random.String(10)
			}
			return files
		}()

		src := filesystem.NewRandom(files, 1024)

		tempDir, err := os.MkdirTemp("", "*")
		if !assertions.Nil(err, "failed create temporary directory") {
			return
		}
		defer os.RemoveAll(tempDir)
		dst := filesystem.NewLocal(tempDir, 0o777, 0o777)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		err = filesystem.CopyWorkers(100, ctx, dst, src)
		if !assertions.Nil(err, "failed to copy files") {
			return
		}
	})
}
