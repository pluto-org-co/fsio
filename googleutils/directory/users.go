// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package directory

import (
	"context"
	"io"
	"iter"
	"log"

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
			OrderBy("email").
			Pages(ctx, func(u *admin.Users) (err error) {
				select {
				case usersCh <- u:
					return nil
				case <-doneCh:
					return io.EOF
				}
			})
		if err != nil {
			log.Println("failed to retrieve users:", err)
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
