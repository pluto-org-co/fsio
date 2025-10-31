package directory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/charlievieth/fastwalk"
	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/ioutils"
)

type Directory struct {
	dirPerm       fs.FileMode
	filePerm      fs.FileMode
	baseDirectory string
	chdir         fs.FS
}

// Creates a new Local Filesystem, Root is the directory only have access to.
// dirPerm corresponds to the permissions used when creating new directories.
// filePerm corresponds to the permissions used when creating a new.
func New(root string, dirPerm, filePerm fs.FileMode) (l *Directory) {
	return &Directory{
		filePerm:      filePerm,
		dirPerm:       dirPerm,
		baseDirectory: path.Clean(root),
		chdir:         os.DirFS(root),
	}
}

var _ filesystem.Filesystem = (*Directory)(nil)

func (l *Directory) ChecksumTime(ctx context.Context, location []string) (checksum string, err error) {
	filename := path.Join(l.baseDirectory, path.Clean(path.Join(location...)))

	info, err := os.Stat(filename)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	checksum = ioutils.ChecksumTime(info.ModTime(), info.Size())
	return checksum, nil
}

func (l *Directory) ChecksumSha256(ctx context.Context, location []string) (checksum string, err error) {
	file, err := l.Open(ctx, location)
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

func (l *Directory) Files(ctx context.Context) (seq iter.Seq[filesystem.FileEntry]) {
	conf := fastwalk.DefaultConfig

	worker := make(chan *filesystem.SimpleFileEntry, 10_000)
	closeCh := make(chan struct{}, 1)
	go func() {
		defer close(worker)

		fastwalk.Walk(&conf, l.baseDirectory, func(fileLocation string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-closeCh:
				return io.EOF
			default:
				info, err := d.Info()
				if err != nil {
					return fmt.Errorf("failed to get file info: %w", err)
				}

				if info.IsDir() || !info.Mode().IsRegular() {
					return nil
				}

				filename, _ := filepath.Rel(l.baseDirectory, fileLocation)

				worker <- &filesystem.SimpleFileEntry{
					LocationValue: strings.Split(filename, "/"),
					ModTimeValue:  info.ModTime(),
				}
				return nil
			}
		})
	}()
	return func(yield func(filesystem.FileEntry) bool) {
		defer func() {
			closeCh <- struct{}{}
			close(closeCh)
		}()
		for entry := range worker {
			if !yield(entry) {
				return
			}
		}
	}
}

func (l *Directory) Open(_ context.Context, location []string) (rc io.ReadCloser, err error) {
	filename := path.Join(location...)
	return l.chdir.Open(filename)
}

func (l *Directory) WriteFile(ctx context.Context, location []string, src io.Reader, modTime time.Time) (finalLocation []string, err error) {
	filename := path.Join(l.baseDirectory, path.Clean(path.Join(location...)))
	filename = path.Clean(filename)

	basedir, _ := path.Split(filename)
	basedir = path.Clean(basedir)

	// Create directory location
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return location, fmt.Errorf("context error during directory creation: %w", err)
		}
		return location, nil
	default:
		err = os.MkdirAll(basedir, l.dirPerm)
		if err != nil {
			return location, fmt.Errorf("failed to create file directory: %w", err)
		}
	}

	// Create file
	var file *os.File
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return location, fmt.Errorf("context error during file creation: %w", err)
		}
		return location, nil
	default:
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, l.filePerm)
		if err != nil {
			return location, fmt.Errorf("failed to create dst file: %w", err)
		}
		defer file.Close()

		// On failure delete the file
		defer func() {
			if err == nil {
				return
			}
			os.Remove(file.Name())
		}()
	}

	switch src.(type) {
	case *bufio.Reader:
		break
	default:
		src = bufio.NewReaderSize(src, ioutils.DefaultBufferSize)
	}

	dst := bufio.NewWriterSize(file, ioutils.DefaultBufferSize)
	defer dst.Flush()

	_, err = ioutils.CopyContext(ctx, dst, src, ioutils.DefaultBufferSize)
	if err != nil {
		return nil, fmt.Errorf("failed to copy contents: %w", err)
	}

	err = dst.Flush()
	if err != nil {
		return nil, fmt.Errorf("failed to flush changes: %w", err)
	}

	err = os.Chtimes(file.Name(), time.Now(), modTime)
	if err != nil {
		return nil, fmt.Errorf("failed to set new mod time: %w", err)
	}
	return location, nil
}

func (l *Directory) RemoveAll(ctx context.Context, location []string) (err error) {
	filename := path.Join(l.baseDirectory, path.Clean(path.Join(location...)))

	return os.Remove(filename)
}

func (l *Directory) Move(ctx context.Context, oldLocation, newLocation []string) (finalLocation []string, err error) {
	oldFilename := path.Join(l.baseDirectory, path.Clean(path.Join(oldLocation...)))
	newFilename := path.Join(l.baseDirectory, path.Clean(path.Join(newLocation...)))

	newDir, _ := path.Split(newFilename)
	err = os.MkdirAll(newDir, l.dirPerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.Rename(oldFilename, newFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to rename file: %w", err)
	}
	return newLocation, nil
}
