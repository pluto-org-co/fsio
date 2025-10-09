package googleutils_test

import (
	"context"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

func Test_SeqDomains(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		conf := creds.NewConfiguration(t, admin.AdminDirectoryDomainReadonlyScope)
		conf.Subject = creds.UserEmail()

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		client := conf.Client(ctx)

		svc, err := admin.NewService(ctx, option.WithHTTPClient(client))
		if !assertions.Nil(err, "failed to create service") {
			return
		}

		for domain := range googleutils.SeqDomains(ctx, svc) {
			t.Logf("Domain: %v", domain.DomainName)
		}
	})
}
