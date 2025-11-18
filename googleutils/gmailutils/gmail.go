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

package gmailutils

import (
	"context"
	"io"
	"iter"
	"log"
	"sync/atomic"

	"google.golang.org/api/gmail/v1"
)

type gmailMailListEntry struct {
	MailList *gmail.ListMessagesResponse
}

func SeqMails(ctx context.Context, svc *gmail.Service) (seq iter.Seq[*gmail.Message]) {
	var done atomic.Bool

	var maillistCh = make(chan *gmailMailListEntry, 1_000)
	go func() {
		defer close(maillistCh)

		err := svc.
			Users.Messages.List("me").
			IncludeSpamTrash(true).
			Pages(ctx, func(lm *gmail.ListMessagesResponse) (err error) {
				if done.Load() {
					return io.EOF
				}

				maillistCh <- &gmailMailListEntry{
					MailList: lm,
				}
				return nil
			})
		if err != nil {
			log.Println("failed to retrieve mails:", err)
			return
		}
	}()

	return func(yield func(*gmail.Message) bool) {
		defer done.Store(true)

		for entry := range maillistCh {
			for _, msg := range entry.MailList.Messages {
				if !yield(msg) {
					return
				}
			}
		}
	}

}
