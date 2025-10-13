package googledrive_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v2"
)

func Test_GoogleDrive(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Test struct {
			Name           string
			CurrentAccount bool
			SharedDrive    bool
			OtherUsers     bool
		}
		var tests = []Test{
			{Name: "Current", CurrentAccount: true},
			{Name: "SharedDrive", SharedDrive: true},
			{Name: "OtherUsers", OtherUsers: true},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				gd := googledrive.New(googledrive.Config{
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
					CurrentAccount: test.CurrentAccount,
					SharedDrive:    test.SharedDrive,
					OtherUsers:     test.OtherUsers,
				})

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

					t.Run(filename, func(t *testing.T) {
						t.Parallel()

						assertions := assert.New(t)

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()

						rd, err := gd.Open(ctx, filename)
						if !assertions.Nil(err, "failed to open file") {
							return
						}
						defer rd.Close()

						hash := sha256.New()
						_, err = io.Copy(hash, bufio.NewReader(rd))
						if !assertions.Nil(err, "failed to hash contents") {
							return
						}

						checksum := hex.EncodeToString(hash.Sum(nil))

						t.Logf("Checksum[%s]: %s", filename, checksum)

					})
				}

				assertions.NotZero(index, "no files found")
			})
		}
	})
}
