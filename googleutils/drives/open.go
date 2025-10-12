package drives

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

// Opens a io.ReadCloser for the filename
func Open(ctx context.Context, svc *drive.Service, filename string) (rc io.ReadCloser, err error) {
	parts := strings.Split(filename, "/")

	var reference *drive.File
	var currentDirectory = "root"
	for index, part := range parts {
		err = svc.Files.
			List().
			Corpora("user").
			PageSize(1).
			Q(fmt.Sprintf("'%s' in parents and name='%s'", currentDirectory, part)).
			Fields("nextPageToken,files(id,name,fullFileExtension,mimeType)").
			Pages(ctx, func(fl *drive.FileList) (err error) {
				if len(fl.Files) == 0 {
					return io.EOF
				}

				currentDirectory = fl.Files[0].Id

				if index == len(parts)-1 {
					reference = fl.Files[0]
				}
				return nil
			})
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}
	}

	// If there is no reference it means the file was not found
	if reference == nil {
		return nil, errors.New("not found")
	}

	file, err := driveutils.Open(ctx, svc, reference.MimeType, reference.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}
