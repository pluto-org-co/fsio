package filesystem_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"iter"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/stretchr/testify/assert"
)

func Test_Local(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		assertions := assert.New(t)

		home, err := os.UserHomeDir()
		if !assertions.Nil(err, "failed to get home directory") {
			return
		}

		root := filesystem.NewLocal(home, 0o777, 0o777)

		t.Run("Files", func(t *testing.T) {
			assertions := assert.New(t)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			var count int

			for range root.Files(ctx) {
				count++
			}
			assertions.NotZero(count, "should found more than one directory")
			t.Logf("Found: %v", count)

			t.Run("EarlyBreak", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
				defer cancel()

				root := filesystem.NewLocal(home, 0o777, 0o777)

				pull, stop := iter.Pull(root.Files(ctx))
				for range 10 {
					_, valid := pull()
					if !valid {
						break
					}
				}
				stop()

				_, valid := pull()
				assertions.False(valid, "should be invalid")
			})
			t.Run("Timeout", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
				defer cancel()

				var timeoutCount int
				root := filesystem.NewLocal(home, 0o777, 0o777)
				for range root.Files(ctx) {
					timeoutCount++
				}
				assertions.Less(timeoutCount, count, "should not found any directory")
			})
		})

		t.Run("Write", func(t *testing.T) {
			assertions := assert.New(t)

			const fileSize = 100 * 1024 * 1024
			src := io.LimitReader(rand.Reader, fileSize)

			checksumHash := sha512.New512_256()
			src = io.TeeReader(src, checksumHash)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			const targetFilename = "sub/location/temporary"

			err = root.WriteFile(ctx, targetFilename, src)
			if !assertions.Nil(err, "failed to write random data to temporary file") {
				return
			}
			defer root.RemoveAll(ctx, targetFilename)

			checksum := hex.EncodeToString(checksumHash.Sum(nil))
			t.Logf("Checksum: %s", checksum)
			t.Run("Check contents", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()
				rc, err := root.Open(ctx, targetFilename)
				if !assertions.Nil(err, "failed to open file") {
					return
				}
				defer rc.Close()

				lastChecksumHash := sha512.New512_256()
				writer := bufio.NewWriter(lastChecksumHash)

				_, err = io.Copy(writer, bufio.NewReader(rc))
				if !assertions.Nil(err, "failed to write to hash") {
					return
				}
				err = writer.Flush()
				if !assertions.Nil(err, "failed to flush pending data") {
					return
				}

				lastChecksum := hex.EncodeToString(lastChecksumHash.Sum(nil))

				assertions.Equal(checksum, lastChecksum, "checksums must match")
			})
			t.Run("Timeout", func(t *testing.T) {
				assertions := assert.New(t)

				openCtx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()
				original, err := root.Open(openCtx, targetFilename)
				if !assertions.Nil(err, "failed to open file") {
					return
				}
				defer original.Close()

				lastChecksumHash := sha512.New512_256()
				src = io.TeeReader(src, lastChecksumHash)

				const targetFilename2 = "sub/location/temporary2"

				writeCtx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
				defer cancel()
				err = root.WriteFile(writeCtx, targetFilename2, src)
				defer root.RemoveAll(ctx, targetFilename2)
				if !assertions.NotNil(err, "should fail to write file") {
					return
				}

				lastChecksum := hex.EncodeToString(lastChecksumHash.Sum(nil))
				if !assertions.NotEqual(checksum, lastChecksum, "checksums should not match") {
					return
				}

			})
		})
	})

}
