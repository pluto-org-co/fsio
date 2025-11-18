// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
