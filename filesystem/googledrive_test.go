package filesystem_test

import (
	"context"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
)

func Test_GoogleDrive(t *testing.T) {
	gd := filesystem.NewGoogleDrive(filesystem.GoogleDriveConfig{
		JWTLoader: func() (config *jwt.Config) {
			config = creds.NewConfiguration(
				t,
				admin.AdminDirectoryUserReadonlyScope,
				admin.AdminDirectoryDomainReadonlyScope,
				drive.DriveScope,
			)
			config.Subject = creds.UserEmail()
			return config
		},
	})

	t.Run("Succeed", func(t *testing.T) {
		t.Run("Files", func(t *testing.T) {
			assertions := assert.New(t)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			var index int
			for filename := range gd.Files(ctx) {
				t.Logf("[%d] Filename: %s", index, filename)
				index++
				if index >= 5 {
					break
				}
			}

			assertions.NotZero(index, "no files found")
		})
	})
}
