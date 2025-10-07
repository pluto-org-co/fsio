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
	"github.com/minio/minio-go/v7"
)

// Generic S3 filesystem
type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(client *minio.Client, bucket string) (s *S3) {
	return &S3{
		client: client,
		bucket: bucket,
	}
}

var _ Filesystem = (*S3)(nil)

func (s *S3) Files(ctx context.Context) (seq iter.Seq[string]) {
	options := minio.ListObjectsOptions{}

	iter := s.client.ListObjectsIter(ctx, s.bucket, options)

	return func(yield func(string) bool) {
		for entry := range iter {
			if entry.Err != nil {
				return
			}

			if !yield(entry.Key) {
				return
			}
		}
	}
}

func (s *S3) Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error) {
	options := minio.GetObjectOptions{}

	obj, err := s.client.GetObject(ctx, s.bucket, filePath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return obj, nil
}

func (s *S3) WriteFile(ctx context.Context, filePath string, src io.Reader) (err error) {
	temp, err := os.CreateTemp("", "*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		temp.Close()
		os.RemoveAll(temp.Name())
	}()

	writer := bufio.NewWriter(temp)
	var reader io.Reader
	switch src.(type) {
	case *bufio.Reader:
		reader = src
	default:
		reader = bufio.NewReader(src)
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
			_, err := io.CopyN(writer, reader, DefaultBufferSize)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return fmt.Errorf("failed to copy chunk: %w", err)
				}
				break loop
			}
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	_, err = temp.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to the begining of the file: %w", err)
	}

	info, err := temp.Stat()
	if err != nil {
		return fmt.Errorf("failed to get temporary file info: %w", err)
	}

	tempReader := bufio.NewReaderSize(temp, DefaultBufferSize)

	var consumedBytes = bytes.NewBuffer(nil)
	mime, err := mimetype.DetectReader(io.TeeReader(tempReader, consumedBytes))
	if err != nil {
		return
	}

	options := minio.PutObjectOptions{
		ContentType: mime.String(),
	}
	_, err = s.client.PutObject(
		ctx,
		s.bucket, filePath,
		io.MultiReader(bytes.NewReader(consumedBytes.Bytes()), tempReader),
		info.Size(),
		options,
	)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

func (s *S3) RemoveAll(ctx context.Context, filePath string) (err error) {
	options := minio.RemoveObjectOptions{}
	return s.client.RemoveObject(ctx, s.bucket, filePath, options)
}
