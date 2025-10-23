package ioutils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func ChecksumSha256(ctx context.Context, src io.Reader) (checksum string, err error) {
	var buffer = bytes.NewBuffer(nil)
	mime, err := mimetype.DetectReader(io.TeeReader(src, buffer))
	if err != nil {
		return "", fmt.Errorf("failed to detect mimetype: %w", err)
	}

	r := io.MultiReader(bytes.NewReader(buffer.Bytes()), src)
	switch {
	case slices.Contains(OfficeMimeTypes, mime.String()):
		temp, err := ReaderToTempFile(ctx, r)
		if err != nil {
			return "", fmt.Errorf("failed to prepare temporary file: %w", err)
		}
		defer temp.Close()

		info, err := temp.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to get file info: %w", err)
		}

		zipReader, err := zip.NewReader(temp, info.Size())
		if err != nil {
			return "", fmt.Errorf("failed to get zip reader: %w", err)
		}

		hash := sha256.New()
		for _, zipFile := range zipReader.File {
			switch {
			case strings.HasPrefix(zipFile.Name, "xl/worksheets/") ||
				strings.HasPrefix(zipFile.Name, "word/media/") ||
				strings.HasPrefix(zipFile.Name, "ppt/slides/") ||
				strings.HasPrefix(zipFile.Name, "Pictures/") || // ODF/OOXML embedded media (common folder name)
				zipFile.Name == "xl/sharedStrings.xml" ||
				zipFile.Name == "xl/workbook.xml" ||
				zipFile.Name == "word/document.xml" ||
				zipFile.Name == "ppt/presentation.xml" ||
				zipFile.Name == "content.xml" || // ODF core document content
				zipFile.Name == "styles.xml" || // ODF core styles (consider content)
				zipFile.Name == "mimetype" || // ODF/OOXML package type
				zipFile.Name == "[Content_Types].xml": // OOXML package type
				err = func() (err error) {
					entryReader, err := zipFile.Open()
					if err != nil {
						return fmt.Errorf("failed to open file: %w", err)
					}
					defer entryReader.Close()

					_, err = CopyContext(ctx, hash, entryReader, DefaultBufferSize)
					if err != nil {
						return fmt.Errorf("failed to copy contents: %w", err)
					}
					return nil
				}()

				if err != nil {
					return "", fmt.Errorf("failed to hash file: %s: %w", zipFile.Name, err)
				}
			default:
				continue
			}
		}

		checksum := hex.EncodeToString(hash.Sum(nil))
		return checksum, nil
	default:
		hash := sha256.New()
		_, err = CopyContext(ctx, hash, bufio.NewReaderSize(r, DefaultBufferSize), DefaultBufferSize)
		if err != nil {
			return "", fmt.Errorf("failed to calculate checksum: %w", err)
		}

		checksum = hex.EncodeToString(hash.Sum(nil))
		return checksum, nil
	}
}
