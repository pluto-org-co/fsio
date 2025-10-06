package filesystem

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"

	"github.com/charlievieth/fastwalk"
)

type Local struct {
	dirPerm       fs.FileMode
	filePerm      fs.FileMode
	baseDirectory string
	chdir         fs.FS
}

// Creates a new Local Filesystem, Root is the directory only have access to.
// dirPerm corresponds to the permissions used when creating new directories.
// filePerm corresponds to the permissions used when creating a new.
func NewLocal(root string, dirPerm, filePerm fs.FileMode) (l *Local) {
	return &Local{
		filePerm:      filePerm,
		dirPerm:       dirPerm,
		baseDirectory: root,
		chdir:         os.DirFS(root),
	}
}

var _ Filesystem = (*Local)(nil)

func (l *Local) Files(ctx context.Context) (seq iter.Seq[string]) {
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

func (l *Local) Open(_ context.Context, filename string) (rc io.ReadCloser, err error) {
	return l.chdir.Open(filename)
}

func (l *Local) WriteFile(ctx context.Context, filePath string, src io.Reader) (err error) {
	filePath = path.Clean(filePath)

	realFilepath := path.Join(l.baseDirectory, filePath)

	dir, _ := path.Split(realFilepath)

	// Create directory location
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return fmt.Errorf("context error during directory creation: %w", err)
		}
		return nil
	default:
		err = os.MkdirAll(dir, l.dirPerm)
		if err != nil {
			return fmt.Errorf("failed to create file directory: %w", err)
		}
	}

	// Create file
	var file *os.File
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err != nil {
			return fmt.Errorf("context error during file creation: %w", err)
		}
		return nil
	default:
		file, err = os.OpenFile(realFilepath, os.O_CREATE|os.O_WRONLY, l.filePerm)
		if err != nil {
			return fmt.Errorf("failed to create dst file: %w", err)
		}
		defer file.Close()

		// On failure delete the file
		defer func() {
			if err == nil {
				return
			}
			os.RemoveAll(file.Name())
		}()
	}

	const DefaultBufferSize = 1024 * 1024 // 1MB
	switch src.(type) {
	case *bufio.Reader:
		break
	default:
		src = bufio.NewReaderSize(src, DefaultBufferSize)
	}

	dst := bufio.NewWriterSize(file, DefaultBufferSize)
	defer dst.Flush()

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return fmt.Errorf("context error during copy: %w", err)
			}
			return nil
		default:
			_, err := io.CopyN(dst, src, DefaultBufferSize)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("failed to copy chunk: %w", err)
			}
		}
	}
}

func (l *Local) RemoveAll(ctx context.Context, filePath string) (err error) {
	filePath = path.Clean(filePath)

	realFilepath := path.Join(l.baseDirectory, filePath)

	return os.RemoveAll(realFilepath)
}
