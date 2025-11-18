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

package filesystem_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/directory"
	"github.com/pluto-org-co/fsio/filesystem/randomfs"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
)

func Test_Sync(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		assertions := assert.New(t)

		files := testsuite.GenerateLocations(100)

		randomSrc := randomfs.New(files, 32*1024*1024)

		srcTmpDir, err := os.MkdirTemp("", "*")
		if !assertions.Nil(err, "failed create temporary directory") {
			return
		}
		defer os.RemoveAll(srcTmpDir)
		src := directory.New(srcTmpDir, 0o777, 0o777)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		err = filesystem.Sync(ctx, src, randomSrc)
		if !assertions.Nil(err, "failed to copy files") {
			return
		}

		t.Run("Sync", func(t *testing.T) {
			assertions := assert.New(t)

			dstTmpDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(dstTmpDir)

			dst := directory.New(dstTmpDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			now := time.Now()
			err = filesystem.Sync(ctx, dst, src)
			if !assertions.Nil(err, "failed to copy files") {
				return
			}
			firstTook := time.Since(now)
			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				now := time.Now()
				err = filesystem.Sync(ctx, dst, src)
				if !assertions.Nil(err, "failed to copy files") {
					return
				}
				secondTook := time.Since(now)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
		})
		t.Run("SyncWorkers", func(t *testing.T) {
			assertions := assert.New(t)

			dstTmpDir, err := os.MkdirTemp("", "*")
			if !assertions.Nil(err, "failed create temporary directory") {
				return
			}
			defer os.RemoveAll(dstTmpDir)

			dst := directory.New(dstTmpDir, 0o777, 0o777)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			now := time.Now()
			err = filesystem.SyncWorkers(100, ctx, dst, src)
			if !assertions.Nil(err, "failed to copy files") {
				return
			}
			firstTook := time.Since(now)

			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				now := time.Now()
				err = filesystem.SyncWorkers(100, ctx, dst, src)
				if !assertions.Nil(err, "failed to copy files") {
					return
				}
				secondTook := time.Since(now)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
		})
	})
}
