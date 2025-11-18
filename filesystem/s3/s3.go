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

package s3

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"strings"
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

func (s *S3) ChecksumTime(ctx context.Context, location []string) (checksum string, err error) {
	objectKey := path.Join(location...)

	options := minio.StatObjectOptions{
		Checksum: true,
	}
	objInfo, err := s.client.StatObject(ctx, s.bucket, objectKey, options)
	if err != nil {
		return "", fmt.Errorf("failed to get object information: %w", err)
	}

	checksum = ioutils.ChecksumTime(LastModifiedFromObj(&objInfo))
	return checksum, nil
}

func (s *S3) ChecksumSha256(ctx context.Context, location []string) (checksum string, err error) {
	objectKey := path.Join(location...)

	options := minio.StatObjectOptions{
		Checksum: true,
	}
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, options)
	if err != nil {
		return "", fmt.Errorf("failed to get object information: %w", err)
	}

	checksum = info.ChecksumSHA256
	if checksum != "" {
		rawChecksum, _ := base64.StdEncoding.DecodeString(checksum)
		return hex.EncodeToString(rawChecksum), nil
	}

	file, err := s.Open(ctx, location)
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

const (
	XAmzMetaMTime   = "X-Amz-Meta-Mtime"
	XAmzCustomMTime = "X-Amz-Custom-Mtime"
)

func LastModifiedFromObj(obj *minio.ObjectInfo) (lastModified time.Time) {
	lastModified = obj.LastModified

	amzMeta, metaFound := obj.UserMetadata[XAmzMetaMTime]
	if metaFound {
		amzMetaTime, err := time.Parse(ioutils.DefaultTimeLayout, amzMeta)
		if err == nil {
			lastModified = amzMetaTime
		} else {
			metaFound = false
		}
	}

	if !metaFound {
		amzCustom, customFound := obj.UserMetadata[XAmzCustomMTime]
		if customFound {
			amzCustomTime, err := time.Parse(ioutils.DefaultTimeLayout, amzCustom)
			if err == nil {
				lastModified = amzCustomTime
			}
		}
	}

	return lastModified
}

func (s *S3) Files(ctx context.Context) (seq iter.Seq[filesystem.FileEntry]) {
	options := minio.ListObjectsOptions{
		WithMetadata: true,
		Recursive:    true,
	}

	objInfoIter := s.client.ListObjectsIter(ctx, s.bucket, options)

	return func(yield func(filesystem.FileEntry) bool) {
		for objInfo := range objInfoIter {
			if objInfo.Err != nil {
				return
			}

			lastModified := LastModifiedFromObj(&objInfo)

			entry := &filesystem.SimpleFileEntry{
				LocationValue: strings.Split(objInfo.Key, "/"),
				ModTimeValue:  lastModified,
			}

			if !yield(entry) {
				return
			}
		}
	}
}

func (s *S3) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	objectKey := path.Join(location...)

	rawFilePathChecksum := sha256.Sum256([]byte(objectKey))
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

	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
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

func (s *S3) WriteFile(ctx context.Context, location []string, src io.Reader, modTime time.Time) (finalLocation []string, err error) {
	objectKey := path.Join(location...)

	srcAsFile, err := ioutils.ReaderToTempFile(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure src is a file: %w", err)
	}
	defer srcAsFile.Close()

	info, err := srcAsFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get temporary file info: %w", err)
	}

	// Compute mimetypes
	mime, err := mimetype.DetectReader(srcAsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to detect mimetype: %w", err)
	}

	_, err = srcAsFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	sTime := modTime.Format(ioutils.DefaultTimeLayout)

	_, err = s.client.PutObject(
		ctx,
		s.bucket, objectKey,
		bufio.NewReaderSize(srcAsFile, ioutils.DefaultBufferSize),
		info.Size(),
		minio.PutObjectOptions{
			ContentType: mime.String(),
			UserMetadata: map[string]string{
				XAmzMetaMTime:   sTime,
				XAmzCustomMTime: sTime,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to put object: %w", err)
	}

	return location, nil
}

func (s *S3) RemoveAll(ctx context.Context, location []string) (err error) {
	objectKey := path.Join(location...)

	options := minio.RemoveObjectOptions{}
	return s.client.RemoveObject(ctx, s.bucket, objectKey, options)
}

func (s *S3) Move(ctx context.Context, oldLocation, newLocation []string) (finalLocation []string, err error) {
	oldObjName := path.Join(oldLocation...)
	newObjName := path.Join(newLocation...)

	dst := minio.CopyDestOptions{Bucket: s.bucket, Object: newObjName}
	src := minio.CopySrcOptions{Bucket: s.bucket, Object: oldObjName}
	_, err = s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	err = s.client.RemoveObject(ctx, s.bucket, oldObjName, minio.RemoveObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to remove old object: %w", err)
	}
	return newLocation, nil
}
