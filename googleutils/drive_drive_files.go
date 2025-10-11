package googleutils

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"path"
	"sync/atomic"
	"time"

	"google.golang.org/api/drive/v2"
)

type gdFileEntry struct {
	path string
	file *drive.File
}

type gdDirEntry struct {
	id       string
	asPrefix string
}

type gdFileListEntry struct {
	dirEntry *gdDirEntry
	filelist *drive.FileList
}

// List all the files in the passed directory using the call as reference factory
func driveFilesFromFilesListCall(ctx context.Context, rootId string, baseCall func() *drive.FilesListCall) (seq iter.Seq2[string, *drive.File]) {
	const MaxTimeouts = 5
	var timeouts int

	var done atomic.Bool
	var doneCh = make(chan struct{}, 1)

	var fileListCh = make(chan *gdFileListEntry, 1_000)
	var filesCh = make(chan *gdFileEntry, 1_000)
	var pendingDirsCh = make(chan *gdDirEntry, 1_000)
	pendingDirsCh <- &gdDirEntry{id: rootId}
	go func() {
		defer func() {
			done.Store(true)
			close(filesCh)
			close(pendingDirsCh)
			close(fileListCh)
		}()
		for {
			select {
			case <-doneCh:
				return
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				timeouts++
				if timeouts >= MaxTimeouts {
					return
				}
			case pendingDir := <-pendingDirsCh:
				timeouts = 0
				// <-workers.Do(
				go func() {
					err := baseCall().
						Q(fmt.Sprintf("'%s' in parents", pendingDir.id)).
						Pages(ctx, func(fl *drive.FileList) (err error) {
							if done.Load() {
								return io.EOF
							}

							fileListCh <- &gdFileListEntry{
								dirEntry: pendingDir,
								filelist: fl,
							}
							return nil
						})
					if err != nil {
						log.Println(err)
						return
					}
				}()
				// )
			case entry := <-fileListCh:
				timeouts = 0
				go func() {
					for _, file := range entry.filelist.Items {
						if done.Load() {
							return
						}

						var filename string = path.Join(entry.dirEntry.asPrefix, file.Title)
						if file.FullFileExtension != "" {
							filename += "." + file.FullFileExtension
						}

						if file.MimeType == "application/vnd.google-apps.folder" {
							pendingDirsCh <- &gdDirEntry{
								id:       file.Id,
								asPrefix: filename,
							}
							continue
						}

						filesCh <- &gdFileEntry{
							path: filename,
							file: file,
						}
					}
				}()
			}
		}
	}()

	return func(yield func(string, *drive.File) bool) {
		defer func() {
			doneCh <- struct{}{}
			close(doneCh)
		}()

		for entry := range filesCh {
			if !yield(entry.path, entry.file) {
				return
			}
		}
	}
}

// List the files of the passed drive
func SeqDriveFiles(ctx context.Context, svc *drive.Service, driveId string) (seq iter.Seq2[string, *drive.File]) {
	return driveFilesFromFilesListCall(ctx, driveId, func() (call *drive.FilesListCall) {
		return svc.Files.
			List().
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			IncludeTeamDriveItems(true).
			Corpora("drive").
			DriveId(driveId)
	})
}
