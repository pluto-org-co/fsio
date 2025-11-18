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
	"iter"
	"log"
	"slices"
	"strings"

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

			log.Println("failed to retrieve domains:", err)
			// TODO: Do something with the error
			return
		}

		slices.SortFunc(domains.Domains, func(a, b *admin.Domains) int { return strings.Compare(a.DomainName, b.DomainName) })

		for _, domain := range domains.Domains {
			if !yield(domain) {
				return
			}
		}
	}
}
