package shareddrives

import (
	"context"
	"iter"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

// List the files of the passed drive
func SeqFiles(ctx context.Context, svc *drive.Service, driveId string) (seq iter.Seq2[[]string, *drive.File]) {
	return driveutils.SeqFilesFromFilesListCall(ctx, driveId, func() (call *drive.FilesListCall) {
		return svc.Files.
			List().
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			IncludeTeamDriveItems(true).
			Corpora("drive").
			DriveId(driveId)
	})
}
