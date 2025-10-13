package shareddrives

import (
	"context"
	"fmt"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

func Checksum(ctx context.Context, svc *drive.Service, driveId, filename string) (checksum string, err error) {
	ref, err := driveutils.FindFileByPath(ctx, filename, driveId, func() *drive.FilesListCall {
		return svc.Files.
			List().
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			IncludeTeamDriveItems(true).
			Corpora("drive").
			DriveId(driveId)
	})
	if err != nil {
		return "", fmt.Errorf("failed to find file: %w", err)
	}

	checksum, err = driveutils.Checksum(ctx, svc, true, ref.Id)
	if err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}
	return checksum, nil
}
