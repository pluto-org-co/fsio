package filesystem

import (
	"context"
	"io"
	"iter"
)

type Filesystem interface {
	// Returns the unique checksum of the file provided
	Checksum(ctx context.Context, filePath string) (checksum string, err error)
	// Returns the seq of all available files in the filesystem
	Files(ctx context.Context) (seq iter.Seq[string])
	// Opens a reader for the passed file.
	Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error)
	// Writes the reader to the dst filePath.
	// Returned filename is the actual name used during the write. Done this way since some implementation may alter the file name during
	// normalization
	WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error)
	// Remove the path from the filesystem
	RemoveAll(ctx context.Context, filePath string) (err error)
}
