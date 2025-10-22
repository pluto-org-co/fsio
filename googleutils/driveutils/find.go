package driveutils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
)

func FindFileByPath(ctx context.Context, filename string, startDirectory string, baseCall func() *drive.FilesListCall) (file *drive.File, err error) {
	parts := strings.Split(filename, "/")

	var currentDirectory = startDirectory
	for index, part := range parts {
		err = baseCall().
			PageSize(1).
			Q(fmt.Sprintf("'%s' in parents and name='%s'", currentDirectory, AddSlashToFilename(part))).
			Fields("nextPageToken,files(id,name,fullFileExtension,mimeType)").
			Pages(ctx, func(fl *drive.FileList) (err error) {
				if len(fl.Files) == 0 {
					return errors.New("no files found - visited files")
				}

				if index == len(parts)-1 {
					file = fl.Files[0]
				} else {
					currentDirectory = fl.Files[0].Id
				}

				return nil
			})
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}
	}

	// If there is no reference it means the file was not found
	if file == nil {
		return nil, errors.New("not found")
	}

	return file, nil
}
