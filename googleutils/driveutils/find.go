package driveutils

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/drive/v3"
)

func FindFileByPath(ctx context.Context, location []string, startDirectory string, baseCall func() *drive.FilesListCall) (file *drive.File, err error) {
	var currentDirectory = startDirectory
	for index, part := range location {
		fl, err := baseCall().
			Q(fmt.Sprintf("trashed=false and '%s' in parents and name='%s'", currentDirectory, part)).
			PageSize(1).
			Fields("nextPageToken,files(id,name,fullFileExtension,mimeType)").
			Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		if len(fl.Files) == 0 {
			return nil, errors.New("no files found - visited files")
		}

		if index == len(location)-1 {
			return fl.Files[0], nil
		} else {
			currentDirectory = fl.Files[0].Id
		}
	}

	// If there is no reference it means the file was not found
	return nil, errors.New("not found")
}
