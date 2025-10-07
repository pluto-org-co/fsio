package filesystem

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/klauspost/compress/gzip"
)

type Gzip struct {
	level int
	fs    Filesystem
}

func NewGzip(level int, fs Filesystem) (g *Gzip) {
	return &Gzip{
		level: level,
		fs:    fs,
	}
}

var _ Filesystem = (*Gzip)(nil)

func (g *Gzip) Files(ctx context.Context) (seq iter.Seq[string]) {
	return g.fs.Files(ctx)
}

type gzipReader struct {
	rc io.ReadCloser
}

func (g *Gzip) Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error) {
	file, err := g.fs.Open(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	var buffer = bytes.NewBuffer(nil)

	mime, err := mimetype.DetectReader(io.TeeReader(file, buffer))
	if err != nil {
		return nil, fmt.Errorf("failed to detect mimetype: %w", err)
	}

	reader := io.MultiReader(bytes.NewReader(buffer.Bytes()), file)
	if mime.Is("application/gzip") {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare gzip reader: %w", err)
		}
	}

	rc = &separateReadCloser{
		closer: file,
		reader: reader,
	}
	return rc, nil
}

func (g *Gzip) WriteFile(ctx context.Context, filePath string, src io.Reader) (err error) {
	rawFile, err := os.CreateTemp("", "*")
	if err != nil {
		return fmt.Errorf("failed to create temporary raw file: %w", err)
	}
	defer rawFile.Close()
	defer os.RemoveAll(rawFile.Name())

	compressedFile, err := os.CreateTemp("", "*")
	if err != nil {
		return fmt.Errorf("failed to create temporary gzip file: %w", err)
	}
	defer compressedFile.Close()
	defer os.RemoveAll(compressedFile.Name())

	// Gzip Writer
	gzipWriter, err := gzip.NewWriterLevel(compressedFile, g.level)
	if err != nil {
		return fmt.Errorf("failed to prepare  gzip writer: %w", err)
	}

	dst := bufio.NewWriterSize(io.MultiWriter(gzipWriter, rawFile), DefaultBufferSize)

	switch src.(type) {
	case *bufio.Reader:
		break
	default:
		src = bufio.NewReaderSize(src, DefaultBufferSize)
	}

loop:
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
				if !errors.Is(err, io.EOF) {
					return fmt.Errorf("failed to copy chunk: %w", err)
				}
				break loop
			}
		}
	}

	err = dst.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush buffered writer: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Write to target
	rawFile.Seek(io.SeekStart, 0)
	compressedFile.Seek(io.SeekStart, 0)

	rawInfo, err := rawFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get raw file info: %w", err)
	}

	compressedInfo, err := compressedFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get compressed file info: %w", err)
	}

	if compressedInfo.Size() < rawInfo.Size() {
		return g.fs.WriteFile(ctx, filePath, compressedFile)
	}

	return g.fs.WriteFile(ctx, filePath, rawFile)
}

func (g *Gzip) RemoveAll(ctx context.Context, filePath string) (err error) {
	return g.fs.RemoveAll(ctx, filePath)
}
