package shareddrives_test

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/shareddrives"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func Test_Checksum(t *testing.T) {
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

						for driveEntry := range shareddrives.SeqDrives(ctx, driveSvc) {
							t.Logf("Drive: %v", driveEntry.Name)
							t.Run(driveEntry.Name, func(t *testing.T) {
								ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
								defer cancel()

								var count int
								for location, file := range shareddrives.SeqFiles(ctx, driveSvc, driveEntry.Id) {
									t.Logf("[%d] File: %s - %v", count, location, file.Id)
									count++
									if count >= 5 {
										break
									}

									t.Run(path.Join(location...), func(t *testing.T) {
										assertions := assert.New(t)

										ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
										defer cancel()
										checksum, err := shareddrives.Checksum(ctx, driveSvc, driveEntry.Id, location)
										if !assertions.Nil(err, "failed to find checksum") {
											return
										}

										t.Logf("Checksum: %s", checksum)
									})
								}
								totalCount += count
							})
							if totalCount >= 10 {
								break
							}
						}
					})
					if totalCount >= 10 {
						break
					}
				}
				if !assertions.NotZero(totalCount, "expecting at least one checksum") {
					return
				}
			})
		}
	})
}
