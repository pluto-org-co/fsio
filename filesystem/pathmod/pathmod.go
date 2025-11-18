// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package pathmod

import (
	"context"
	"io"
	"iter"
	"time"

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

func (p *PathMod) ChecksumTime(ctx context.Context, location []string) (checksum string, err error) {
	return p.fs.ChecksumTime(ctx, location)
}

func (p *PathMod) ChecksumSha256(ctx context.Context, location []string) (checksum string, err error) {
	return p.fs.ChecksumSha256(ctx, location)
}

func (p *PathMod) Files(ctx context.Context) (seq iter.Seq[filesystem.FileEntry]) {
	return p.fs.Files(ctx)
}

func (p *PathMod) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	return p.fs.Open(ctx, location)
}

func (p *PathMod) WriteFile(ctx context.Context, location []string, src io.Reader, modTime time.Time) (finalLocation []string, err error) {
	return p.fs.WriteFile(ctx, p.f(location), src, modTime)
}

func (p *PathMod) RemoveAll(ctx context.Context, location []string) (err error) {
	return p.fs.RemoveAll(ctx, location)
}

func (p *PathMod) Move(ctx context.Context, oldLocation, newLocation []string) (finalLocation []string, err error) {
	return p.fs.Move(ctx, oldLocation, newLocation)
}

func New(fs filesystem.Filesystem, f PathModFunc) (p *PathMod) {
	return &PathMod{
		fs: fs,
		f:  f,
	}
}
