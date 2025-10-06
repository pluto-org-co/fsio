package filesystem

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
)

// Random read filesystem
// Used for testing the performance of the overall system
type Random struct {
	locations map[string]struct{}
	fileSizes int64
}

func NewRandom(locations []string, fileSizes int64) (r *Random) {
	r = &Random{
		locations: make(map[string]struct{}, len(locations)),
		fileSizes: fileSizes,
	}
	for _, location := range locations {
		r.locations[location] = struct{}{}
	}
	return r
}

var _ Filesystem = (*Random)(nil)

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

	rc = io.NopCloser(bufio.NewReader(io.LimitReader(rand.Reader, r.fileSizes)))
	return rc, nil
}

func (r *Random) WriteFile(ctx context.Context, filePath string, src io.Reader) (err error) {
	const DefaultChunkSize = 1024 * 1024
	dst := bufio.NewWriterSize(io.Discard, DefaultChunkSize)
	defer dst.Flush()

	src = bufio.NewReaderSize(src, DefaultChunkSize)

	for {
		_, err = io.CopyN(dst, src, DefaultChunkSize)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("failed to copy chunk: %w", err)
		}
	}
}

func (r *Random) RemoveAll(ctx context.Context, filePath string) (err error) {
	delete(r.locations, filePath)
	return nil
}
