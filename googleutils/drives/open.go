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
