package filesystem

import (
	"context"
	"fmt"
	"sync"
)

func Copy(ctx context.Context, dst, src Filesystem) (err error) {
	for filename := range src.Files(ctx) {
		err = func() (err error) {
			file, err := src.Open(ctx, filename)
			if err != nil {
				return fmt.Errorf("failed to open src file: %w", err)
			}
			defer file.Close()

			_, err = dst.WriteFile(ctx, filename, file)
			if err != nil {
				return fmt.Errorf("failed to write dst file: %w", err)
			}

			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to copy: %s: %w", filename, err)
		}
	}
	return nil
}

func CopyWorkers(workersNumber int, ctx context.Context, dst, src Filesystem) (err error) {
	workers := make(chan struct{}, workersNumber)
	for range workersNumber {
		workers <- struct{}{}
	}

	errorsCh := make(chan error, workersNumber)
	defer close(errorsCh)

	var wg sync.WaitGroup
	defer wg.Wait()
	for filename := range src.Files(ctx) {
		select {
		case <-workers:
			wg.Add(1)
			go func() {
				defer func() {
					workers <- struct{}{}
					wg.Done()
				}()
				err = func() (err error) {
					file, err := src.Open(ctx, filename)
					if err != nil {
						return fmt.Errorf("failed to open src file: %w", err)
					}
					defer file.Close()

					_, err = dst.WriteFile(ctx, filename, file)
					if err != nil {
						return fmt.Errorf("failed to write dst file: %w", err)
					}

					return nil
				}()
				if err != nil {
					errorsCh <- fmt.Errorf("failed to copy: %s: %w", filename, err)
				}
			}()
		case err = <-errorsCh:
			return fmt.Errorf("errors during copy: %w", err)
		}
	}
	return nil
}
