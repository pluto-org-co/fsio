package directory_test

import (
	"context"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

func Test_SeqUsers(t *testing.T) {
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

		for domain := range directory.SeqDomains(ctx, svc) {
			t.Logf("Domain: %s", domain.DomainName)

			t.Run(domain.DomainName, func(t *testing.T) {
				assertions := assert.New(t)

				var count int
				for u := range directory.SeqUsers(ctx, svc, domain.DomainName) {
					t.Logf("User: %v", u.PrimaryEmail)
					count++
				}
				if !assertions.NotZero(count, "expecting at least one user") {
					return
				}
			})
		}
	})
}
