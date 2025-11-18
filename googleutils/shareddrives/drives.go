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
	"iter"
	"log"
	"slices"
	"strings"

	"google.golang.org/api/drive/v3"
)

// List the drives that the account can access:
// Requires at least: https://www.googleapis.com/auth/drive
func SeqDrives(ctx context.Context, svc *drive.Service) (seq iter.Seq[*drive.Drive]) {
	const MaxPageSize = 100

	return func(yield func(*drive.Drive) bool) {
		var drives = make([]*drive.Drive, 0, MaxPageSize)
		err := svc.Drives.
			List().
			Context(ctx).
			PageSize(MaxPageSize).
			Pages(ctx, func(dl *drive.DriveList) (err error) {
				drives = append(drives, dl.Drives...)
				return nil
			})
		if err != nil {
			log.Println("failed to retrieve drives:", err)
			// TODO: What should this do with the error?
			return
		}

		slices.SortFunc(drives, func(a, b *drive.Drive) int { return strings.Compare(a.Name, b.Name) })

		for _, drive := range drives {
			if !yield(drive) {
				return
			}
		}
	}
}
