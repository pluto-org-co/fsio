package pathmod

import (
	"context"
	"io"
	"iter"

	"github.com/pluto-org-co/fsio/filesystem"
)

type PathModFunc func(oldLocation []string) (newLocation []string)

// This FS modifies the path passed to the underlying Filesystem based on a modification function.
// Listing files works as normal. The path modification is performed for every WriteFile.
type PathMod struct {
	fs filesystem.Filesystem
	f  PathModFunc
}

var _ filesystem.Filesystem = (*PathMod)(nil)

func (p *PathMod) Checksum(ctx context.Context, location []string) (checksum string, err error) {
	return p.fs.Checksum(ctx, location)
}

func (p *PathMod) Files(ctx context.Context) (seq iter.Seq[[]string]) {
	return p.fs.Files(ctx)
}

func (p *PathMod) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	return p.fs.Open(ctx, location)
}

func (p *PathMod) WriteFile(ctx context.Context, location []string, src io.Reader) (finalLocation []string, err error) {
	return p.fs.WriteFile(ctx, p.f(location), src)
}

func (p *PathMod) RemoveAll(ctx context.Context, location []string) (err error) {
	return p.fs.RemoveAll(ctx, location)
}

func New(fs filesystem.Filesystem, f PathModFunc) (p *PathMod) {
	return &PathMod{
		fs: fs,
		f:  f,
	}
}
