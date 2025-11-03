package gmailutils_test

import (
	"context"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/gmailutils"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func Test_SeqMails(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		conf := creds.NewConfiguration(t, googleutils.Scopes...)
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

				var totalCount int
				for u := range directory.SeqUsers(ctx, svc, domain.DomainName) {
					t.Run(u.PrimaryEmail, func(t *testing.T) {
						assertions := assert.New(t)

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()

						conf := creds.NewConfiguration(t, googleutils.Scopes...)
						conf.Subject = u.PrimaryEmail

						gmailSvc, err := gmail.NewService(ctx, option.WithHTTPClient(conf.Client(ctx)))
						if !assertions.Nil(err, "failed to get gmail service") {
							return
						}

						var count int
						for mail := range gmailutils.SeqMails(ctx, gmailSvc) {
							t.Logf("Contents: %v", mail.Id)
							count++
							if count >= 5 {
								break
							}
						}
						totalCount += count
					})

					if totalCount >= 5 {
						break
					}
				}
				if !assertions.NotZero(totalCount, "expecting at least one user") {
					return
				}
			})
		}
	})
}
