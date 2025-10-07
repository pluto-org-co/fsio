package filesystem

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/pluto-org-co/fsio/ioutils"
)

type SelfdestructionFile struct {
	*os.File
}

func (f *SelfdestructionFile) Close() (err error) {
	f.File.Close()
	os.Remove(f.File.Name())
	return nil
}

// Save a temporary file in order to prevent connection timeout or similar error from unknown readers.
// This functions creates the temporary file and automatically cleans on close.
// If reader is actually a *os.File, the function will return it without managing its deletion
func ReaderToTempFile(ctx context.Context, src io.Reader) (file fs.File, err error) {
	switch tr := src.(type) {
	case *os.File:
		return tr, nil
	default:
		temp, err := os.CreateTemp("", "*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file: %w", err)
		}
		defer func() {
			if err != nil {
				temp.Close()
				os.Remove(temp.Name())
			}
		}()
		go func() {
			<-ctx.Done()
			temp.Close()
			os.Remove(temp.Name())
		}()

		writer := bufio.NewWriter(temp)
		var reader io.Reader
		switch src.(type) {
		case *bufio.Reader:
			reader = src
		default:
			reader = bufio.NewReader(src)
		}

		_, err = ioutils.CopyContext(ctx, writer, reader, DefaultBufferSize)
		if err != nil {
			return nil, fmt.Errorf("failed to copy contents: %w", err)
		}

		err = writer.Flush()
		if err != nil {
			return nil, fmt.Errorf("failed to flush writer: %w", err)
		}

		_, err = temp.Seek(0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to the begining of the file: %w", err)
		}

		return &SelfdestructionFile{File: temp}, nil
	}
}
