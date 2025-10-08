package filesystem

import (
	"context"
	"io"
	"iter"
)

type PathModFunc func(oldNew string) (newPath string)

// This FS modifies the path passed to the underlying Filesystem based on a modification function.
// Listing files works as normal. The path modification is performed for every WriteFile.
type PathMod struct {
	fs Filesystem
	f  PathModFunc
}

var _ Filesystem = (*PathMod)(nil)

func (p *PathMod) Files(ctx context.Context) (seq iter.Seq[string]) {
	return p.fs.Files(ctx)
}

func (p *PathMod) Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error) {
	return p.fs.Open(ctx, filePath)
}

func (p *PathMod) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	return p.fs.WriteFile(ctx, p.f(filePath), src)
}

func (p *PathMod) RemoveAll(ctx context.Context, filePath string) (err error) {
	return p.fs.RemoveAll(ctx, filePath)
}

func NewPathMod(fs Filesystem, f PathModFunc) (p *PathMod) {
	return &PathMod{
		fs: fs,
		f:  f,
	}
}
