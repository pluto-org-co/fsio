package driveutils

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/api/drive/v3"
)

type gdFileEntry struct {
	path []string
	file *drive.File
}

type gdDirEntry struct {
	id       string
	asPrefix []string
}

type gdFileListEntry struct {
	dirEntry *gdDirEntry
	filelist *drive.FileList
}

// List all the files in the passed directory using the call as reference factory
func SeqFilesFromFilesListCall(ctx context.Context, rootId string, baseCall func() *drive.FilesListCall) (seq iter.Seq2[[]string, *drive.File]) {
	const MaxTimeouts = 25
	var timeouts int

	var done atomic.Bool
	var doneCh = make(chan struct{}, 1)

	var fileListCh = make(chan *gdFileListEntry, 1_000)
	var filesCh = make(chan *gdFileEntry, 1_000)
	var pendingDirsCh = make(chan *gdDirEntry, 1_000)
	pendingDirsCh <- &gdDirEntry{id: rootId}

	go func() {
		var wg sync.WaitGroup
		defer func() {
			wg.Wait()
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
			case <-time.After(100 * time.Millisecond):
				timeouts++
				if timeouts >= MaxTimeouts {
					return
				}
			case pendingDir := <-pendingDirsCh:
				timeouts = 0
				wg.Go(func() {
					err := baseCall().
						PageSize(1_000).
						Q(fmt.Sprintf("trashed=false and '%s' in parents", pendingDir.id)).
						Fields("nextPageToken,files(id,name,fullFileExtension,mimeType,modifiedTime)").
						OrderBy("name").
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
						return
					}
				})
			case entry := <-fileListCh:
				timeouts = 0
				wg.Go(func() {
					defer func() {
						if err := recover(); err != nil {

							log.Println("failed to retrieve files:", err)
							// TODO: Log on DEV builds
						}
					}()
					for _, file := range entry.filelist.Files {
						if done.Load() {
							return
						}

						location := append(slices.Clone(entry.dirEntry.asPrefix), file.Name)
						if file.MimeType == "application/vnd.google-apps.folder" {
							pendingDirsCh <- &gdDirEntry{
								id:       file.Id,
								asPrefix: location,
							}
							continue
						}

						filesCh <- &gdFileEntry{
							path: location,
							file: file,
						}
					}
				})
			}
		}
	}()

	return func(yield func([]string, *drive.File) bool) {
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
