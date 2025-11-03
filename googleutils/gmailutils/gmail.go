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

	var mailCh = make(chan *gmailMailListEntry, 1_000)
	go func() {
		err := svc.
			Users.Messages.List("me").
			IncludeSpamTrash(true).
			Pages(ctx, func(lm *gmail.ListMessagesResponse) (err error) {
				if done.Load() {
					return io.EOF
				}

				mailCh <- &gmailMailListEntry{
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

		for entry := range mailCh {
			for _, msg := range entry.MailList.Messages {
				if !yield(msg) {
					return
				}
			}
		}
	}

}
