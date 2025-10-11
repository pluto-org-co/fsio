package shareddrives

import (
	"context"
	"io"
	"iter"

	"google.golang.org/api/drive/v2"
)

// List the drives that the account can access:
// Requires at least: https://www.googleapis.com/auth/drive
func SeqDrives(ctx context.Context, svc *drive.Service) (seq iter.Seq[*drive.Drive]) {
	var doneCh = make(chan struct{}, 1)
	var drivesCh = make(chan *drive.DriveList, 1_000)

	go func() {
		defer close(drivesCh)

		err := svc.Drives.
			List().
			Context(ctx).
			Pages(ctx, func(dl *drive.DriveList) (err error) {
				select {
				case <-doneCh:
					return io.EOF
				case drivesCh <- dl:
					return nil
				}
			})
		if err != nil {
			// TODO: What should this do with the error?
		}
	}()
	return func(yield func(*drive.Drive) bool) {
		defer close(doneCh)

		for driveList := range drivesCh {
			for _, drive := range driveList.Items {
				if !yield(drive) {
					doneCh <- struct{}{}
					return
				}
			}
		}
	}
}
