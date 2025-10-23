package drives_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/drives"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func Test_Open(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		conf := creds.NewConfiguration(t, admin.AdminDirectoryUserReadonlyScope, admin.AdminDirectoryDomainReadonlyScope)
		conf.Subject = creds.UserEmail()

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		client := conf.Client(ctx)

		adminSvc, err := admin.NewService(ctx, option.WithHTTPClient(client))
		if !assertions.Nil(err, "failed to create service") {
			return
		}

		for domain := range directory.SeqDomains(ctx, adminSvc) {
			t.Logf("Domain: %s", domain.DomainName)

			t.Run(domain.DomainName, func(t *testing.T) {
				assertions := assert.New(t)
				var totalCount int
				for u := range directory.SeqUsers(ctx, adminSvc, domain.DomainName) {
					t.Run(u.PrimaryEmail, func(t *testing.T) {
						assertions := assert.New(t)

						conf := creds.NewConfiguration(t,
							admin.AdminDirectoryUserReadonlyScope,
							admin.AdminDirectoryDomainReadonlyScope,
							gdrive.DriveScope,
						)
						conf.Subject = u.PrimaryEmail

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()
						client := conf.Client(ctx)

						driveSvc, err := gdrive.NewService(ctx, option.WithHTTPClient(client))
						if !assertions.Nil(err, "failed to create service") {
							return
						}

						var count int
						for location, file := range drives.SeqFiles(ctx, driveSvc) {
							t.Logf("[%d] File: %s - %v", count, location, file.Id)
							t.Run(path.Join(location...), func(t *testing.T) {
								assertions := assert.New(t)

								ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
								defer cancel()
								rc, err := drives.Open(ctx, driveSvc, location)
								if !assertions.Nil(err, "failed to open filename") {
									return
								}
								defer rc.Close()

								hash := sha256.New()
								_, err = io.Copy(hash, bufio.NewReader(rc))
								if !assertions.Nil(err, "failed to hash file") {
									return
								}

								checksum := hex.EncodeToString(hash.Sum(nil))
								t.Logf("Checksum[%s]: %s", location, checksum)
							})
							count++
							if count >= 5 {
								break
							}
						}
						totalCount += count
					})
					if totalCount >= 100 {
						break
					}
				}
				assertions.NotZero(totalCount, "no files found")
			})
		}
	})
}
