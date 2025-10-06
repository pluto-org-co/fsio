package testsuite

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

func TestFilesystem(t *testing.T, baseFs filesystem.Filesystem) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("Succeed", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Can't run this test as root")
				return
			}

			const TotalFiles = 1_000
			testFs := baseFs

			t.Run("Files", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				var count int
				for range testFs.Files(ctx) {
					count++
				}
				assertions.Equal(TotalFiles, count, "should found the expected number of files")
				t.Logf("Found: %v", count)

				t.Run("EarlyBreak", func(t *testing.T) {
					assertions := assert.New(t)

					ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
					defer cancel()

					pull, stop := iter.Pull(testFs.Files(ctx))
					for range 10 {
						_, valid := pull()
						if !valid {
							break
						}
					}
					stop()

					_, valid := pull()
					assertions.False(valid, "should be invalid after stop()")
				})
				t.Run("Timeout", func(t *testing.T) {
					assertions := assert.New(t)

					ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
					defer cancel()

					var timeoutCount int
					for range testFs.Files(ctx) {
						timeoutCount++
					}
					assertions.Less(timeoutCount, count, "should not find all files due to timeout")
				})
			})

			t.Run("Write", func(t *testing.T) {
				assertions := assert.New(t)

				const fileSize = 100 * 1024 * 1024
				randSrc := io.LimitReader(rand.Reader, fileSize)

				checksumHash := sha512.New512_256()
				randSrc = io.TeeReader(randSrc, checksumHash)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				const targetFilename = "sub/location/temporary"

				err := testFs.WriteFile(ctx, targetFilename, randSrc)
				if !assertions.Nil(err, "failed to write random data to temporary file") {
					return
				}
				defer testFs.RemoveAll(ctx, targetFilename)

				checksum := hex.EncodeToString(checksumHash.Sum(nil))
				t.Logf("Checksum: %s", checksum)

				t.Run("Check contents", func(t *testing.T) {
					assertions := assert.New(t)

					ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
					defer cancel()

					rc, err := testFs.Open(ctx, targetFilename)
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
					original, err := testFs.Open(openCtx, targetFilename)
					if !assertions.Nil(err, "failed to open file for second write test") {
						return
					}
					defer original.Close()

					lastChecksumHash := sha512.New512_256()
					randSrc := io.TeeReader(original, lastChecksumHash)

					const targetFilename2 = "sub/location/temporary2"

					writeCtx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
					defer cancel()
					err = testFs.WriteFile(writeCtx, targetFilename2, randSrc)
					defer testFs.RemoveAll(ctx, targetFilename2)

					if !assertions.NotNil(err, "should fail to write file due to short timeout") {
						return
					}

					lastChecksum := hex.EncodeToString(lastChecksumHash.Sum(nil))
					if !assertions.NotEqual(checksum, lastChecksum, "checksums should not match because the write was incomplete/timed out") {
						return
					}

				})
			})
		})
	}
}
