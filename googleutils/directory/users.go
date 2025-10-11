package directory

import (
	"context"
	"io"
	"iter"

	admin "google.golang.org/api/admin/directory/v1"
)

// Helper function for iterating over all accounts in the domain
// This requires at least: https://www.googleapis.com/auth/admin.directory.user.readonly
func SeqUsers(ctx context.Context, svc *admin.Service, domain string) (seq iter.Seq[*admin.User]) {
	var doneCh = make(chan struct{}, 1)
	var usersCh = make(chan *admin.Users, 1_000)

	go func() {
		defer close(usersCh)

		err := svc.Users.
			List().
			Context(ctx).
			Domain(domain).
			Pages(ctx, func(u *admin.Users) (err error) {
				select {
				case usersCh <- u:
					return nil
				case <-doneCh:
					return io.EOF
				}
			})
		if err != nil {
			// TODO: What should this do with the error?
		}
	}()
	return func(yield func(*admin.User) bool) {
		defer close(doneCh)

		for users := range usersCh {
			for _, user := range users.Users {
				if !yield(user) {
					doneCh <- struct{}{}
					return
				}
			}
		}
	}
}
