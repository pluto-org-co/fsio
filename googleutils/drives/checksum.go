package drives

import (
	"context"
	"fmt"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

func Checksum(ctx context.Context, svc *drive.Service, filename string) (checksum string, err error) {
	ref, err := driveutils.FindFileByPath(ctx, filename, "root", func() *drive.FilesListCall {
		return svc.Files.List().Corpora("user")
	})
	if err != nil {
		return "", fmt.Errorf("failed to find file: %w", err)
	}

	checksum, err = driveutils.Checksum(ctx, svc, false, ref.Id)
	if err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}
	return checksum, nil
}
