package syncutils

type Workers struct {
	available chan struct{}
}

func (w *Workers) Get() (rCh <-chan struct{}) {
	return w.available
}

func (w *Workers) Put() {
	w.available <- struct{}{}
}

func (w *Workers) Do(f func()) {
	defer recover()
	<-w.available
	go func() {
		defer func() {
			recover()
			w.available <- struct{}{}
		}()

		f()
	}()
}

func (w *Workers) Close() (err error) {
	close(w.available)
	return nil
}

func NewWorkers(n int) (w *Workers) {
	w = &Workers{available: make(chan struct{}, n)}
	for range n {
		w.available <- struct{}{}
	}
	return w
}
