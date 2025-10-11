package filesystem

import (
	"context"
	"fmt"

	"github.com/pluto-org-co/fsio/syncutils"
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
	workers := syncutils.NewWorkers(workersNumber)
	defer workers.Wait()

	errorsCh := make(chan error, workersNumber)
	defer close(errorsCh)

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
		case <-workers.Do(func() {
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
		}):
			continue
		}
	}

	return nil
}
