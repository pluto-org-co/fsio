package googleutils_test

import (
	"context"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

func Test_SeqUserDrive(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		conf := creds.NewConfiguration(t, admin.AdminDirectoryUserReadonlyScope, admin.AdminDirectoryDomainReadonlyScope)
		conf.Subject = creds.UserEmail()

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		client := conf.Client(ctx)

		svc, err := admin.NewService(ctx, option.WithHTTPClient(client))
		if !assertions.Nil(err, "failed to create service") {
			return
		}

		for domain := range googleutils.SeqDomains(ctx, svc) {
			t.Logf("Domain: %s", domain.DomainName)

			t.Run(domain.DomainName, func(t *testing.T) {
				assertions := assert.New(t)
				var count int
				for u := range googleutils.SeqUsers(ctx, svc, domain.DomainName) {
					t.Run(u.PrimaryEmail, func(t *testing.T) {
						assertions := assert.New(t)

						conf := creds.NewConfiguration(t,
							admin.AdminDirectoryUserReadonlyScope,
							admin.AdminDirectoryDomainReadonlyScope,
							drive.DriveScope,
						)
						conf.Subject = u.PrimaryEmail

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()
						client := conf.Client(ctx)

						svc, err := drive.NewService(ctx, option.WithHTTPClient(client))
						if !assertions.Nil(err, "failed to create service") {
							return
						}

						for drive := range googleutils.SeqUserDrives(ctx, svc) {
							t.Logf("Drive: %v", drive.Name)
							count++
						}
					})
					if count > 0 {
						break
					}
				}
				if !assertions.NotZero(count, "expecting at least one drive") {
					return
				}
			})
		}
	})
}
