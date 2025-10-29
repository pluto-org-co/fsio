package filesystem_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
)

func Test_SyncDrive(t *testing.T) {
	const DriveTimeout = 10 * time.Second
	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		src := googledrive.New(googledrive.Config{
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
			CurrentAccount: true,
			SharedDrive:    true,
			OtherUsers:     true,
		})

		t.Run("Sync", func(t *testing.T) {
			assertions := assert.New(t)

			dstTmpDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(dstTmpDir)

			dst := directory.New(dstTmpDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), DriveTimeout)
			defer cancel()

			now := time.Now()
			filesystem.Sync(ctx, dst, src)
			firstTook := time.Since(now)
			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), DriveTimeout)
				defer cancel()

				now := time.Now()
				filesystem.Sync(ctx, dst, src)
				secondTook := time.Since(now)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
		})
		t.Run("SyncWorkers", func(t *testing.T) {
			assertions := assert.New(t)

			dstTmpDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(dstTmpDir)

			dst := directory.New(dstTmpDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), DriveTimeout)
			defer cancel()

			now := time.Now()
			filesystem.SyncWorkers(100, ctx, dst, src)
			firstTook := time.Since(now)

			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), DriveTimeout)
				defer cancel()

				now := time.Now()
				filesystem.SyncWorkers(100, ctx, dst, src)
				secondTook := time.Since(now)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
		})
	})
}
