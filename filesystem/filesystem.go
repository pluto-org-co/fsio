package filesystem

import (
	"context"
	"io"
	"iter"
)

const DefaultBufferSize = 1024 * 1024

type Filesystem interface {
	// Returns the seq of all available files in the filesystem
	Files(ctx context.Context) (seq iter.Seq[string])
	// Opens a reader for the passed file.
	Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error)
	// Writes the reader to the dst filename
	WriteFile(ctx context.Context, filePath string, src io.Reader) (err error)
	// Remove the path from the filesystem
	RemoveAll(ctx context.Context, filePath string) (err error)
}
