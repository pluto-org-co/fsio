package googleutils

import (
	"context"
	"iter"

	admin "google.golang.org/api/admin/directory/v1"
)

// List all the domains managed by the account.
// It uses "my_customer" as passed value for the list function.
// Requires: https://www.googleapis.com/auth/admin.directory.domain.readonly
func SeqDomains(ctx context.Context, svc *admin.Service) (seq iter.Seq[*admin.Domains]) {
	return func(yield func(*admin.Domains) bool) {
		domains, err := svc.Domains.
			List("my_customer").
			Do()
		if err != nil {
			// TODO: Do something with the error
			return
		}

		for _, domain := range domains.Domains {
			if !yield(domain) {
				return
			}
		}
	}
}
