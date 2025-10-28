package gzipfs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/klauspost/compress/gzip"
	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/utils"
	"github.com/pluto-org-co/fsio/ioutils"
)

type Gzip struct {
	level int
	fs    filesystem.Filesystem
}

func New(level int, fs filesystem.Filesystem) (g *Gzip) {
	return &Gzip{
		level: level,
		fs:    fs,
	}
}

var _ filesystem.Filesystem = (*Gzip)(nil)

func (g *Gzip) ChecksumSha256(ctx context.Context, location []string) (checksum string, err error) {
	file, err := g.Open(ctx, location)
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

func (g *Gzip) Files(ctx context.Context) (seq iter.Seq[[]string]) {
	return g.fs.Files(ctx)
}

type gzipReader struct {
	rc io.ReadCloser
}

func (g *Gzip) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	file, err := g.fs.Open(ctx, location)
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

	rc = utils.NewSeparateReadCloser(file, reader)
	return rc, nil
}

func (g *Gzip) WriteFile(ctx context.Context, location []string, src io.Reader) (finalLocation []string, err error) {
	rawFile, err := os.CreateTemp("", "*")
	if err != nil {
		return location, fmt.Errorf("failed to create temporary raw file: %w", err)
	}
	defer rawFile.Close()
	defer os.Remove(rawFile.Name())

	compressedFile, err := os.CreateTemp("", "*")
	if err != nil {
		return location, fmt.Errorf("failed to create temporary gzip file: %w", err)
	}
	defer compressedFile.Close()
	defer os.Remove(compressedFile.Name())

	// Gzip Writer
	gzipWriter, err := gzip.NewWriterLevel(compressedFile, g.level)
	if err != nil {
		return location, fmt.Errorf("failed to prepare  gzip writer: %w", err)
	}

	dst := bufio.NewWriterSize(io.MultiWriter(gzipWriter, rawFile), ioutils.DefaultBufferSize)

	switch src.(type) {
	case *bufio.Reader:
		break
	default:
		src = bufio.NewReaderSize(src, ioutils.DefaultBufferSize)
	}

	_, err = ioutils.CopyContext(ctx, dst, src, ioutils.DefaultBufferSize)
	if err != nil {
		return location, fmt.Errorf("failed to copy contents: %w", err)
	}

	err = dst.Flush()
	if err != nil {
		return location, fmt.Errorf("failed to flush buffered writer: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return location, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Write to target
	rawFile.Seek(io.SeekStart, 0)
	compressedFile.Seek(io.SeekStart, 0)

	rawInfo, err := rawFile.Stat()
	if err != nil {
		return location, fmt.Errorf("failed to get raw file info: %w", err)
	}

	compressedInfo, err := compressedFile.Stat()
	if err != nil {
		return location, fmt.Errorf("failed to get compressed file info: %w", err)
	}

	if compressedInfo.Size() < rawInfo.Size() {
		return g.fs.WriteFile(ctx, location, compressedFile)
	}

	return g.fs.WriteFile(ctx, location, rawFile)
}

func (g *Gzip) RemoveAll(ctx context.Context, location []string) (err error) {
	return g.fs.RemoveAll(ctx, location)
}
