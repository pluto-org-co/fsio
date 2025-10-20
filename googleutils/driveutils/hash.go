package driveutils

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/pluto-org-co/fsio/ioutils"
	"google.golang.org/api/drive/v3"
)

func Checksum(ctx context.Context, svc *drive.Service, driveFile bool, fileId string) (checksum string, err error) {
	getCall := svc.Files.
		Get(fileId).
		Context(ctx).
		SupportsAllDrives(true).
		SupportsTeamDrives(true).
		Fields("id,name,mimeType,sha256Checksum")
	if driveFile {
		getCall = getCall.SupportsAllDrives(true).SupportsTeamDrives(true)
	}

	reference, err := getCall.Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file by id: %w", err)
	}

	if reference.Sha256Checksum != "" {
		return reference.Sha256Checksum, nil
	}

	file, err := Open(ctx, svc, reference.MimeType, reference.Id)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %s: %w", reference.Name, err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = ioutils.CopyContext(ctx, hash, bufio.NewReaderSize(file, ioutils.DefaultBufferSize), ioutils.DefaultBufferSize)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum = hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}
