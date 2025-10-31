package filesystem

import (
	"context"
	"io"
	"iter"
	"time"
)

type SimpleFileEntry struct {
	LocationValue []string
	ModTimeValue  time.Time
}

var _ FileEntry = (*SimpleFileEntry)(nil)

func (f *SimpleFileEntry) Location() (location []string) {
	return f.LocationValue
}

func (f *SimpleFileEntry) ModTime() (mtime time.Time) {
	return f.ModTimeValue
}

type FileEntry interface {
	Location() (location []string)
	ModTime() (mtime time.Time)
}

type Filesystem interface {
	// Returns the unique time checksum of the file provided
	ChecksumTime(ctx context.Context, location []string) (checksum string, err error)
	// Returns the unique sha256 checksum of the file provided
	ChecksumSha256(ctx context.Context, location []string) (checksum string, err error)
	// Returns the seq of all available files in the filesystem
	Files(ctx context.Context) (seq iter.Seq[FileEntry])
	// Opens a reader for the passed file.
	Open(ctx context.Context, location []string) (rc io.ReadCloser, err error)
	// Writes the reader to the dst filePath.
	// Returned filename is the actual name used during the write. Done this way since some implementation may alter the file name during
	// normalization
	WriteFile(ctx context.Context, location []string, src io.Reader, modTime time.Time) (finalLocation []string, err error)
	// Remove the path from the filesystem
	RemoveAll(ctx context.Context, location []string) (err error)
}
