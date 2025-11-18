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
	"fmt"
	"slices"
	"time"

	"github.com/pluto-org-co/fsio/ioutils"
	"google.golang.org/api/drive/v3"
)

func ChecksumTime(ctx context.Context, svc *drive.Service, driveFile bool, fileId string) (checksum string, err error) {
	getCall := svc.Files.
		Get(fileId).
		Context(ctx).
		SupportsAllDrives(true).
		SupportsTeamDrives(true).
		Fields("id,name,mimeType,sha256Checksum,modifiedTime,size")
	if driveFile {
		getCall = getCall.SupportsAllDrives(true).SupportsTeamDrives(true)
	}

	info, err := getCall.Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file by id: %w", err)
	}

	modTime, err := time.Parse(time.RFC3339, info.ModifiedTime)
	if err != nil {
		return "", fmt.Errorf("failed to parse modifiedTime: %w", err)
	}

	checksum = ioutils.ChecksumTime(modTime)
	return checksum, nil
}

func ChecksumSha256(ctx context.Context, svc *drive.Service, driveFile bool, fileId string) (checksum string, err error) {
	getCall := svc.Files.
		Get(fileId).
		Context(ctx).
		SupportsAllDrives(true).
		SupportsTeamDrives(true).
		Fields("id,name,mimeType,sha256Checksum,modifiedTime,size")
	if driveFile {
		getCall = getCall.SupportsAllDrives(true).SupportsTeamDrives(true)
	}

	info, err := getCall.Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file by id: %w", err)
	}

	if info.Sha256Checksum != "" && !slices.Contains(ioutils.OfficeLikeMimeTypes, info.MimeType) {
		return info.Sha256Checksum, nil
	}

	file, err := Open(ctx, svc, info.MimeType, info.Id)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %s: %w", info.Name, err)
	}
	defer file.Close()

	checksum, err = ioutils.ChecksumSha256(ctx, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return checksum, nil
}
