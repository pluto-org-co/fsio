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
	return func(yield func(string) bool) {
		fs.WalkDir(l.chdir, ".", func(dirPath string, d fs.DirEntry, err error) (errFinal error) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if d.IsDir() {
					return nil
				}

				filename := path.Join(dirPath, d.Name())

				if !yield(filename) {
					return io.EOF
				}
				return nil
			}

		})
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
