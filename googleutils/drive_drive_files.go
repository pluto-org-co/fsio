package googleutils

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"path"
	"sync"
	"sync/atomic"

	"google.golang.org/api/drive/v2"
)

type dirEntry struct {
	id       string
	asPrefix string
}

type fileEntry struct {
	dirEntry *dirEntry
	filelist *drive.FileList
}

// List all the files in the passed directory using the call as reference factory
func recursiveDriveFiles(ctx context.Context, baseCall func() *drive.FilesListCall) (seq iter.Seq2[string, *drive.File]) {
	var done atomic.Bool

	var pendingDirsCh = make(chan *dirEntry, 1_000)
	pendingDirsCh <- &dirEntry{id: "root"}

	var fileListCh = make(chan *fileEntry, 1_000)
	go func() {
		defer func() {
			close(pendingDirsCh)
			close(fileListCh)
		}()
		for {
			select {
			case pendingDir := <-pendingDirsCh:
				func() {
					var wg sync.WaitGroup
					defer wg.Wait()

					// First query directories
					wg.Go(func() {
						err := baseCall().
							Context(ctx).
							Q(fmt.Sprintf("'%s' in parents and mimeType='application/vnd.google-apps.folder'", pendingDir.id)).
							Pages(ctx, func(fl *drive.FileList) error {
								if done.Load() {
									return io.EOF
								}

								go func() {
									for _, directory := range fl.Items {
										var dirname = directory.Title
										if directory.FullFileExtension != "" {
											dirname += "." + directory.FullFileExtension
										}

										pendingDirsCh <- &dirEntry{id: directory.Id, asPrefix: path.Join(pendingDir.asPrefix, dirname)}
									}
								}()
								return nil
							})
						if err != nil {
							log.Println(err)
							return
						}
					})

					// Queries files too
					wg.Go(func() {
						err := baseCall().
							Context(ctx).
							Q(fmt.Sprintf("'%s' in parents and mimeType!='application/vnd.google-apps.folder'", pendingDir.id)).
							Pages(ctx, func(fl *drive.FileList) error {
								if done.Load() {
									return io.EOF
								}

								fileListCh <- &fileEntry{dirEntry: pendingDir, filelist: fl}
								return nil
							})

						if err != nil {
							done.Store(true)
							log.Println(err)
							return
						}
					})
				}()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(yield func(string, *drive.File) bool) {
		defer done.Store(true)

		for entry := range fileListCh {
			for _, file := range entry.filelist.Items {
				var filename = file.Title
				if file.FullFileExtension != "" {
					filename += "." + file.FullFileExtension
				}

				if !yield(path.Join(entry.dirEntry.asPrefix, filename), file) {
					return
				}
			}

		}
	}
}

// List the files of the passed drive
func SeqDriveFiles(ctx context.Context, svc *drive.Service, driveId string) (seq iter.Seq2[string, *drive.File]) {
	return recursiveDriveFiles(ctx, func() (call *drive.FilesListCall) {
		return svc.Files.
			List().
			DriveId(driveId).
			IncludeItemsFromAllDrives(true).
			SupportsAllDrives(true).
			Corpora("drive")
	})
}
