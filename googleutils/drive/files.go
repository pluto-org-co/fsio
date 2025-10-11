package drives

import (
	"context"
	"iter"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v2"
)

// List the files accessible in the users owned drive
func SeqFiles(ctx context.Context, svc *drive.Service) (seq iter.Seq2[string, *drive.File]) {
	return driveutils.SeqFilesFromFilesListCall(ctx, "root", func() (call *drive.FilesListCall) { return svc.Files.List().Corpora("default") })
}
