package directory

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"

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
		baseDirectory: root,
		chdir:         os.DirFS(root),
	}
}

var _ filesystem.Filesystem = (*Directory)(nil)

func (l *Directory) ChecksumSha256(ctx context.Context, filePath string) (checksum string, err error) {
	file, err := l.Open(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = ioutils.CopyContext(ctx, hash, bufio.NewReaderSize(file, ioutils.DefaultBufferSize), ioutils.DefaultBufferSize)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))

	return checksum, nil
}

func (l *Directory) Files(ctx context.Context) (seq iter.Seq[string]) {
	conf := fastwalk.DefaultConfig

	worker := make(chan string, 1_000)
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

				worker <- filename
				return nil
			}
		})
	}()
	return func(yield func(string) bool) {
		defer func() {
			closeCh <- struct{}{}
			close(closeCh)
		}()
		for filename := range worker {
			if !yield(filename) {
				return
			}
		}
	}
}

func (l *Directory) Open(_ context.Context, filename string) (rc io.ReadCloser, err error) {
	return l.chdir.Open(filename)
}

func (l *Directory) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	filePath = path.Clean(filePath)

	realFilepath := path.Join(l.baseDirectory, filePath)

	dir, _ := path.Split(realFilepath)

	// Create directory location
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return filePath, fmt.Errorf("context error during directory creation: %w", err)
		}
		return filePath, nil
	default:
		err = os.MkdirAll(dir, l.dirPerm)
		if err != nil {
			return filePath, fmt.Errorf("failed to create file directory: %w", err)
		}
	}

	// Create file
	var file *os.File
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return filePath, fmt.Errorf("context error during file creation: %w", err)
		}
		return filePath, nil
	default:
		file, err = os.OpenFile(realFilepath, os.O_CREATE|os.O_WRONLY, l.filePerm)
		if err != nil {
			return filePath, fmt.Errorf("failed to create dst file: %w", err)
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
		return filePath, fmt.Errorf("failed to copy contents: %w", err)
	}
	return filePath, nil
}

func (l *Directory) RemoveAll(ctx context.Context, filePath string) (err error) {
	filePath = path.Clean(filePath)

	realFilepath := path.Join(l.baseDirectory, filePath)

	return os.Remove(realFilepath)
}
