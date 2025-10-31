package filesystem

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type SyncCtx struct {
	MaxFiles int64
}

type SyncOption func(ctx *SyncCtx) (err error)

func WithSyncOptionMaxFiles(maxFiles int64) (option SyncOption) {
	return func(ctx *SyncCtx) (err error) {
		ctx.MaxFiles = maxFiles
		return nil
	}
}

// Same as Copy but doesn't stop on errors
func Sync(ctx context.Context, dst, src Filesystem, options ...SyncOption) (err error) {
	var syncCtx = &SyncCtx{
		MaxFiles: -1,
	}
	for _, option := range options {
		option(syncCtx)
	}

	now := time.Now()

	var count int64
	for entry := range src.Files(ctx) {
		if syncCtx.MaxFiles > 0 && count >= syncCtx.MaxFiles {
			return nil
		}
		count++

		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return fmt.Errorf("failed to sync do to context error: %w", err)
			}
			return nil
		default:
			err := func() (err error) {
				srcChecksum, _ := src.ChecksumTime(ctx, entry.Location())
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
				err = fmt.Errorf("failed to sync: %s: %w", entry, err)
				log.Println(err)
				if errors.Is(err, context.DeadlineExceeded) {
					return err
				}
			}
		}

	}
	return nil
}

func SyncWorkers(workersNumber int, ctx context.Context, dst, src Filesystem, options ...SyncOption) (err error) {
	var syncCtx = &SyncCtx{
		MaxFiles: 0,
	}
	for _, option := range options {
		option(syncCtx)
	}

	now := time.Now()

	var workers = make(chan struct{}, workersNumber)
	for range workersNumber {
		workers <- struct{}{}
	}
	defer close(workers)

	errorsCh := make(chan error, workersNumber)

	var wg sync.WaitGroup
	defer wg.Wait()

	var count int64
	for entry := range src.Files(ctx) {
		if syncCtx.MaxFiles > 0 && count >= syncCtx.MaxFiles {
			return nil
		}
		count++

		select {
		case err := <-errorsCh:
			if errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			log.Println(err)
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return fmt.Errorf("failed to sync do to context error: %w", err)
			}
			return nil
		case <-workers:
			wg.Go(func() {
				defer func() { workers <- struct{}{} }()

				err := func() (err error) {
					srcChecksum, _ := src.ChecksumTime(ctx, entry.Location())
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
					errorsCh <- fmt.Errorf("failed to sync: %s: %w", entry, err)
				}
			})
		}
	}

	return nil
}
