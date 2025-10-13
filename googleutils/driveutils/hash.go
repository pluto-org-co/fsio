package driveutils

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/pluto-org-co/fsio/ioutils"
	"google.golang.org/api/drive/v3"
)

func Checksum(ctx context.Context, svc *drive.Service, driveFile bool, fileId string) (checksum string, err error) {
	getCall := svc.Files.
		Get(fileId).
		Context(ctx).
		Fields("id,name,mimeType,md5Checksum,sha1Checksum,sha256Checksum")
	if driveFile {
		getCall = getCall.SupportsAllDrives(true).SupportsTeamDrives(true)
	}

	reference, err := getCall.Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file by id: %w", err)
	}

	checksum = reference.Md5Checksum + reference.Sha1Checksum + reference.Sha256Checksum
	if checksum != "" {
		return checksum, nil
	}

	file, err := Open(ctx, svc, reference.MimeType, reference.Id)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %s: %w", reference.Name, err)
	}
	defer file.Close()

	hash := sha512.New512_256()
	_, err = ioutils.CopyContext(ctx, hash, bufio.NewReaderSize(file, ioutils.DefaultBufferSize), ioutils.DefaultBufferSize)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}
