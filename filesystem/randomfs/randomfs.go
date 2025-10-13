package randomfs

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/ioutils"
	"github.com/pluto-org-co/fsio/random"
)

// Random read filesystem
// Used for testing the performance of the overall system
type Random struct {
	locations map[string]struct{}
	fileSizes int64
}

func New(locations []string, fileSizes int64) (r *Random) {
	r = &Random{
		locations: make(map[string]struct{}, len(locations)),
		fileSizes: fileSizes,
	}
	for _, location := range locations {
		r.locations[location] = struct{}{}
	}
	return r
}

var _ filesystem.Filesystem = (*Random)(nil)

func (r *Random) Checksum(ctx context.Context, filePath string) (checksum string, err error) {
	file, err := r.Open(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha512.New512_256()
	_, err = ioutils.CopyContext(ctx, hash, bufio.NewReaderSize(file, ioutils.DefaultBufferSize), ioutils.DefaultBufferSize)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))

	return checksum, nil
}

func (r *Random) Files(ctx context.Context) (seq iter.Seq[string]) {
	return func(yield func(string) bool) {
		for location := range r.locations {
			select {
			case <-ctx.Done():
				return
			default:
				if !yield(location) {
					return
				}
			}
		}
	}
}

func (r *Random) Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error) {
	_, found := r.locations[filePath]
	if !found {
		return nil, os.ErrNotExist
	}

	rc = io.NopCloser(bufio.NewReader(io.LimitReader(random.InsecureReader, r.fileSizes)))
	return rc, nil
}

func (r *Random) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	r.locations[filePath] = struct{}{}

	dst := bufio.NewWriterSize(io.Discard, ioutils.DefaultBufferSize)
	defer dst.Flush()

	src = bufio.NewReaderSize(src, ioutils.DefaultBufferSize)

	for {
		_, err = io.CopyN(dst, src, ioutils.DefaultBufferSize)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return filePath, nil
			}
			return filePath, fmt.Errorf("failed to copy chunk: %w", err)
		}
	}
}

func (r *Random) RemoveAll(ctx context.Context, filePath string) (err error) {
	delete(r.locations, filePath)
	return nil
}
