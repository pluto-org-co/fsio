package filesystem

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func Copy(ctx context.Context, dst, src Filesystem) (err error) {
	now := time.Now()

	for entry := range src.Files(ctx) {
		err = func() (err error) {
			srcChecksum, err := src.ChecksumTime(ctx, entry.Location())
			if err != nil {
				return fmt.Errorf("failed to get src checksum: %w", err)
			}

			dstChecksum, _ := dst.ChecksumTime(ctx, entry.Location())
			if srcChecksum != "" && srcChecksum == dstChecksum {
				return nil
			}

			file, err := src.Open(ctx, entry.Location())
			if err != nil {
				return fmt.Errorf("failed to open src file: %w", err)
			}
			defer file.Close()

			_, err = dst.WriteFile(ctx, entry.Location(), file, now)
			if err != nil {
				return fmt.Errorf("failed to write dst file: %w", err)
			}

			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to copy: %s: %w", entry, err)
		}
	}
	return nil
}

func CopyWorkers(workersNumber int, ctx context.Context, dst, src Filesystem) (err error) {
	now := time.Now()

	var workers = make(chan struct{}, workersNumber)
	for range workersNumber {
		workers <- struct{}{}
	}
	defer close(workers)

	errorsCh := make(chan error, workersNumber)

	var wg sync.WaitGroup
	defer wg.Wait()

	for entry := range src.Files(ctx) {
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
					srcChecksum, err := src.ChecksumTime(ctx, entry.Location())
					if err != nil {
						return fmt.Errorf("failed to get src checksum: %w", err)
					}

					dstChecksum, _ := dst.ChecksumTime(ctx, entry.Location())
					if srcChecksum != "" && srcChecksum == dstChecksum {
						return nil
					}

					srcFile, err := src.Open(ctx, entry.Location())
					if err != nil {
						return fmt.Errorf("failed to open src file: %w", err)
					}
					defer srcFile.Close()

					_, err = dst.WriteFile(ctx, entry.Location(), srcFile, now)
					if err != nil {
						return fmt.Errorf("failed to write dst file: %w", err)
					}

					return nil
				}()
				if err != nil {
					errorsCh <- fmt.Errorf("failed to copy: %s: %w", entry, err)
				}
			})
		}
	}

	return nil
}
