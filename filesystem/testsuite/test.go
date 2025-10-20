package testsuite

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"iter"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/randomfs"
	"github.com/pluto-org-co/fsio/ioutils"
	"github.com/pluto-org-co/fsio/random"
	"github.com/stretchr/testify/assert"
)

func TestFilesystem(t *testing.T, baseFs filesystem.Filesystem) func(t *testing.T) {
	assertions := assert.New(t)

	files := GenerateFilenames(100)

	randomRoot := randomfs.New(files, 32*1024*1024)

	ctxCopy, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	err := filesystem.CopyWorkers(100, ctxCopy, baseFs, randomRoot)
	if !assertions.Nil(err, "failed to copy fs contents") {
		return func(t *testing.T) {}
	}

	return func(t *testing.T) {
		t.Run("Succeed", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Can't run this test as root")
				return
			}

			testFs := baseFs

			t.Run("Files", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				var count int
				for range testFs.Files(ctx) {
					count++
				}
				assertions.NotZero(count, "should found the expected number of files")
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
				randSrc := io.LimitReader(random.InsecureReader, fileSize)

				referenceChecksumHash := sha256.New()
				counter := ioutils.NewCountWriter(referenceChecksumHash)
				randSrc = io.TeeReader(randSrc, counter)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				targetFilename, err := testFs.WriteFile(ctx, GenerateFilename(5), randSrc)
				if !assertions.Nil(err, "failed to write random data to temporary file") {
					return
				}
				defer testFs.RemoveAll(ctx, targetFilename)

				t.Logf("WritFile Bytes count: %d", counter.Count())

				referenceChecksum := hex.EncodeToString(referenceChecksumHash.Sum(nil))
				t.Logf("Reference Checksum: %s", referenceChecksum)

				fsChecksum, err := testFs.Checksum(ctx, targetFilename)
				if !assertions.Nil(err, "failed to compute file checksum") {
					return
				}

				if !assertions.NotEmpty(fsChecksum, "failed to request file checksum") {
					return
				}
				t.Logf("FS Checksum: %s", fsChecksum)

				if !assertions.Equal(fsChecksum, referenceChecksum, "checksums doesn't match") {
					return
				}

				t.Run("Check contents", func(t *testing.T) {
					assertions := assert.New(t)

					ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
					defer cancel()

					rc, err := testFs.Open(ctx, targetFilename)
					if !assertions.Nil(err, "failed to open file") {
						return
					}
					defer rc.Close()

					computedChecksum := sha256.New()
					counter := ioutils.NewCountWriter(computedChecksum)
					writer := bufio.NewWriter(counter)

					_, err = io.Copy(writer, bufio.NewReader(rc))
					if !assertions.Nil(err, "failed to write to hash") {
						return
					}
					err = writer.Flush()
					if !assertions.Nil(err, "failed to flush pending data") {
						return
					}

					t.Logf("Last checksum writes: %d", counter.Count())
					readChecksum := hex.EncodeToString(computedChecksum.Sum(nil))

					assertions.Equal(referenceChecksum, readChecksum, "checksums must match")
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

					lastChecksumHash := sha256.New()
					randSrc := io.TeeReader(original, lastChecksumHash)

					var targetFilename2 = GenerateFilename(5)

					writeCtx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
					defer cancel()
					_, err = testFs.WriteFile(writeCtx, targetFilename2, randSrc)
					defer testFs.RemoveAll(ctx, targetFilename2)

					if !assertions.NotNil(err, "should fail to write file due to short timeout") {
						return
					}

					lastChecksum := hex.EncodeToString(lastChecksumHash.Sum(nil))
					if !assertions.NotEqual(referenceChecksum, lastChecksum, "checksums should not match because the write was incomplete/timed out") {
						return
					}
				})
			})
		})
	}
}
