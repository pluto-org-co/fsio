package randomfs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"strings"
	"time"

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

func New(locations [][]string, fileSizes int64) (r *Random) {
	r = &Random{
		locations: make(map[string]struct{}, len(locations)),
		fileSizes: fileSizes,
	}
	for _, location := range locations {
		r.locations[path.Join(location...)] = struct{}{}
	}
	return r
}

var _ filesystem.Filesystem = (*Random)(nil)

func (r *Random) ChecksumTime(ctx context.Context, location []string) (checksum string, err error) {
	return ioutils.ChecksumTime(time.Now(), int64(random.InsecureReader.Uint64())), nil
}

func (r *Random) ChecksumSha256(ctx context.Context, location []string) (checksum string, err error) {
	file, err := r.Open(ctx, location)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	checksum, err = ioutils.ChecksumSha256(ctx, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}
	return checksum, nil
}

func (r *Random) Files(ctx context.Context) (seq iter.Seq[filesystem.FileEntry]) {
	return func(yield func(filesystem.FileEntry) bool) {
		for location := range r.locations {
			select {
			case <-ctx.Done():
				return
			default:
				if !yield(&filesystem.SimpleFileEntry{
					LocationValue: strings.Split(location, "/"),
					ModTimeValue:  time.Date(2005, 01, 01, 01, 0, 0, 0, time.UTC),
				}) {
					return
				}
			}
		}
	}
}

func (r *Random) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	_, found := r.locations[path.Join(location...)]
	if !found {
		return nil, os.ErrNotExist
	}

	rc = io.NopCloser(bufio.NewReader(io.LimitReader(random.InsecureReader, r.fileSizes)))
	return rc, nil
}

func (r *Random) WriteFile(ctx context.Context, location []string, src io.Reader, modTime time.Time) (finalLocation []string, err error) {
	filename := path.Join(location...)
	r.locations[filename] = struct{}{}

	dst := bufio.NewWriterSize(io.Discard, ioutils.DefaultBufferSize)
	defer dst.Flush()

	src = bufio.NewReaderSize(src, ioutils.DefaultBufferSize)

	for {
		_, err = io.CopyN(dst, src, ioutils.DefaultBufferSize)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return location, nil
			}
			return location, fmt.Errorf("failed to copy chunk: %w", err)
		}
	}
}

func (r *Random) RemoveAll(ctx context.Context, location []string) (err error) {
	delete(r.locations, path.Join(location...))
	return nil
}

func (r *Random) Move(ctx context.Context, oldLocation, newLocation []string) (finalLocation []string, err error) {
	delete(r.locations, path.Join(oldLocation...))
	r.locations[path.Join(newLocation...)] = struct{}{}
	return newLocation, nil
}
