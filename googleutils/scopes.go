package googleutils

import (
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
)

var Scopes = []string{
	admin.AdminDirectoryUserReadonlyScope,
	admin.AdminDirectoryDomainReadonlyScope,
	drive.DriveScope,
	gmail.GmailReadonlyScope,
}
