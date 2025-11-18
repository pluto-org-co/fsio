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

package shareddrives

import (
	"context"
	"fmt"

	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"google.golang.org/api/drive/v3"
)

func ChecksumTime(ctx context.Context, svc *drive.Service, driveId string, location []string) (checksum string, err error) {
	ref, err := driveutils.FindFileByPath(ctx, location, driveId, func() *drive.FilesListCall {
		return svc.Files.
			List().
			SupportsAllDrives(true).
			SupportsTeamDrives(true).
			IncludeItemsFromAllDrives(true).
			IncludeTeamDriveItems(true).
			Corpora("drive").
			DriveId(driveId)
	})
	if err != nil {
		return "", fmt.Errorf("failed to find file: %w", err)
	}

	checksum, err = driveutils.ChecksumTime(ctx, svc, true, ref.Id)
	if err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}
	return checksum, nil
}

func ChecksumSha256(ctx context.Context, svc *drive.Service, driveId string, location []string) (checksum string, err error) {
	ref, err := driveutils.FindFileByPath(ctx, location, driveId, func() *drive.FilesListCall {
		return svc.Files.
			List().
			SupportsAllDrives(true).
			SupportsTeamDrives(true).
			IncludeItemsFromAllDrives(true).
			IncludeTeamDriveItems(true).
			Corpora("drive").
			DriveId(driveId)
	})
	if err != nil {
		return "", fmt.Errorf("failed to find file: %w", err)
	}

	checksum, err = driveutils.ChecksumSha256(ctx, svc, true, ref.Id)
	if err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}
	return checksum, nil
}
