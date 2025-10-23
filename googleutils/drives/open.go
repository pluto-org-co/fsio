package drives

import (
	"context"
	"fmt"
	"io"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

// Opens a io.ReadCloser for the filename
func Open(ctx context.Context, svc *drive.Service, location []string) (rc io.ReadCloser, err error) {
	reference, err := driveutils.FindFileByPath(ctx, location, "root", func() *drive.FilesListCall {
		return svc.Files.List().Corpora("user")
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find file: %w", err)
	}

	file, err := driveutils.Open(ctx, svc, reference.MimeType, reference.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}
