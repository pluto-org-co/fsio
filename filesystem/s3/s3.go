package s3

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/ioutils"
)

// Generic S3 filesystem
type S3 struct {
	client      *minio.Client
	bucket      string
	cacheExpiry time.Duration
}

func New(client *minio.Client, bucket string, cacheExpiry time.Duration) (s *S3) {
	return &S3{
		client:      client,
		bucket:      bucket,
		cacheExpiry: cacheExpiry,
	}
}

var _ filesystem.Filesystem = (*S3)(nil)

func (s *S3) Checksum(ctx context.Context, filePath string) (checksum string, err error) {
	options := minio.StatObjectOptions{
		Checksum: true,
	}
	info, err := s.client.StatObject(ctx, s.bucket, filePath, options)
	if err != nil {
		return "", fmt.Errorf("failed to get object information: %w", err)
	}

	checksum = info.ETag + info.ChecksumCRC32 + info.ChecksumCRC32C + info.ChecksumCRC64NVME + info.ChecksumMode + info.ChecksumSHA1 + info.ChecksumSHA256
	if checksum != "" {
		return checksum, nil
	}

	file, err := s.Open(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha512.New512_256()
	_, err = ioutils.CopyContext(ctx, hash, bufio.NewReaderSize(file, ioutils.DefaultBufferSize), ioutils.DefaultBufferSize)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}

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
	rawFilePathChecksum := sha512.Sum512_256([]byte(filePath))
	filePathChecksum := hex.EncodeToString(rawFilePathChecksum[:])

	cachedFilePath := path.Join(os.TempDir(), filePathChecksum)

	cachedFile, err := os.Open(cachedFilePath)
	if err == nil {
		time.AfterFunc(s.cacheExpiry, func() {
			cachedFile.Close()
			os.Remove(cachedFilePath)
		})
		return cachedFile, nil
	}

	if os.IsExist(err) {
		os.Remove(cachedFilePath)
		return nil, fmt.Errorf("failed to open cached file: %w", err)
	}

	cachedFile, err = os.Create(cachedFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err != nil {
			cachedFile.Close()
			os.Remove(cachedFilePath)
		}
	}()

	obj, err := s.client.GetObject(ctx, s.bucket, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	writer := bufio.NewWriterSize(cachedFile, ioutils.DefaultBufferSize)
	reader := bufio.NewReaderSize(obj, ioutils.DefaultBufferSize)

	_, err = ioutils.CopyContext(ctx, writer, reader, ioutils.DefaultBufferSize)
	if err != nil {
		return nil, fmt.Errorf("failed to copy contents: %w", err)
	}

	err = writer.Flush()
	if err != nil {
		return nil, fmt.Errorf("failed to flush writer: %w", err)
	}

	_, err = cachedFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to reset file cursor: %w", err)
	}
	time.AfterFunc(s.cacheExpiry, func() {
		cachedFile.Close()
		os.Remove(cachedFilePath)
	})

	return cachedFile, nil
}

func (s *S3) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	srcAsFile, err := ioutils.ReaderToTempFile(ctx, src)
	if err != nil {
		return filePath, fmt.Errorf("failed to ensure src is a file: %w", err)
	}
	defer srcAsFile.Close()

	info, err := srcAsFile.Stat()
	if err != nil {
		return filePath, fmt.Errorf("failed to get temporary file info: %w", err)
	}

	reader := bufio.NewReaderSize(srcAsFile, ioutils.DefaultBufferSize)

	var consumedBytes = bytes.NewBuffer(nil)
	mime, err := mimetype.DetectReader(io.TeeReader(reader, consumedBytes))
	if err != nil {
		return "", fmt.Errorf("failed to detect mimetype: %w", err)
	}

	_, err = s.client.PutObject(
		ctx,
		s.bucket, filePath,
		io.MultiReader(bytes.NewReader(consumedBytes.Bytes()), reader),
		info.Size(),
		minio.PutObjectOptions{
			ContentType:  mime.String(),
			AutoChecksum: minio.ChecksumCRC32,
		},
	)

	if err != nil {
		return filePath, fmt.Errorf("failed to put object: %w", err)
	}

	return filePath, nil
}

func (s *S3) RemoveAll(ctx context.Context, filePath string) (err error) {
	options := minio.RemoveObjectOptions{}
	return s.client.RemoveObject(ctx, s.bucket, filePath, options)
}
