package shareddrives

import (
	"context"
	"iter"
	"log"
	"slices"
	"strings"

	"google.golang.org/api/drive/v3"
)

// List the drives that the account can access:
// Requires at least: https://www.googleapis.com/auth/drive
func SeqDrives(ctx context.Context, svc *drive.Service) (seq iter.Seq[*drive.Drive]) {
	const MaxPageSize = 100

	return func(yield func(*drive.Drive) bool) {
		var drives = make([]*drive.Drive, 0, MaxPageSize)
		err := svc.Drives.
			List().
			Context(ctx).
			PageSize(MaxPageSize).
			Pages(ctx, func(dl *drive.DriveList) (err error) {
				drives = append(drives, dl.Drives...)
				return nil
			})
		if err != nil {
			log.Println("failed to retrieve drives:", err)
			// TODO: What should this do with the error?
			return
		}

		slices.SortFunc(drives, func(a, b *drive.Drive) int { return strings.Compare(a.Name, b.Name) })

		for _, drive := range drives {
			if !yield(drive) {
				return
			}
		}
	}
}
