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
	var workers = make(chan struct{}, workersNumber)
	for range workersNumber {
		workers <- struct{}{}
	}
	defer close(workers)

	errorsCh := make(chan error, workersNumber)

	var wg sync.WaitGroup
	defer wg.Wait()

	for filename := range src.Files(ctx) {
		select {
		case err = <-errorsCh:
			return fmt.Errorf("errors during copy: %w", err)
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return fmt.Errorf("context error: %w", err)
			}
			return nil
		case <-workers:
			wg.Go(func() {
				defer func() { workers <- struct{}{} }()

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
			})
		}
	}

	return nil
}
