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

package googleutils

import (
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
)

// API Permissions required to use all the googleutils implemented
// Notice this will also require to enable the APIs from the Cloud Console
// - Admin SDK API
// - Drive API
// - Gmail API
var Scopes = []string{
	admin.AdminDirectoryUserReadonlyScope,
	admin.AdminDirectoryDomainReadonlyScope,
	drive.DriveScope,
	gmail.GmailReadonlyScope,
}
