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

package filesystem

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"sync"
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

				srcFile, err := src.Open(ctx, entry.Location())
				if err != nil {
					return fmt.Errorf("failed to open src file: %w", err)
				}
				defer srcFile.Close()

				_, err = dst.WriteFile(ctx, entry.Location(), srcFile, entry.ModTime())
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
						log.Println("SKIP:", path.Join(entry.Location()...))
						return nil
					}

					srcFile, err := src.Open(ctx, entry.Location())
					if err != nil {
						return fmt.Errorf("failed to open src file: %w", err)
					}
					defer srcFile.Close()

					_, err = dst.WriteFile(ctx, entry.Location(), srcFile, entry.ModTime())
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
