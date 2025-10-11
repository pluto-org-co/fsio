package syncutils

import (
	"log"
	"sync"
)

type Workers struct {
	wg        sync.WaitGroup
	available chan struct{}
}

func (w *Workers) Do(f func()) (rCh <-chan struct{}) {
	out := make(chan struct{})

	w.wg.Go(func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("ERROR", err)
			}

			// In case w.available is closed we need to recover
			// This will prevent sending a new job panic
			select {
			case w.available <- struct{}{}:
			default:
				// This means the worker pool was closed
				return
			}
		}()

		// Wait for available jobs
		<-w.available

		// Notify caller the job was taken
		out <- struct{}{}
		close(out)

		f()
	})
	return out
}

func (w *Workers) Wait() {
	w.wg.Wait()
}

func (w *Workers) Close() (err error) {
	close(w.available)
	w.wg.Wait()
	return nil
}

func NewWorkers(n int) (w *Workers) {
	w = &Workers{
		available: make(chan struct{}, n),
	}
	for range n {
		w.available <- struct{}{}
	}
	return w
}
