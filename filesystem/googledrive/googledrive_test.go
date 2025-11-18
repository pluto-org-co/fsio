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

package googledrive_test

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/ioutils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/jwt"
)

func Test_GoogleDrive(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Test struct {
			Name           string
			CurrentAccount bool
			SharedDrive    bool
			OtherUsers     bool
		}
		var tests = []Test{
			{Name: "Current", CurrentAccount: true},
			{Name: "SharedDrive", SharedDrive: true},
			{Name: "OtherUsers", OtherUsers: true},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				gd := googledrive.New(googledrive.Config{
					JWTLoader: func() (config *jwt.Config) {
						config = creds.NewConfiguration(
							t, googleutils.Scopes...,
						)
						config.Subject = creds.UserEmail()
						return config
					},
					CurrentAccount: test.CurrentAccount,
					SharedDrive:    test.SharedDrive,
					OtherUsers:     test.OtherUsers,
				})

				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				var index int
				for entry := range gd.Files(ctx) {
					t.Logf("[%d] Filename: %s", index, entry)
					index++
					if index >= 5 {
						break
					}

					t.Run(path.Join(entry.Location()...), func(t *testing.T) {
						assertions := assert.New(t)

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()

						rd, err := gd.Open(ctx, entry.Location())
						if !assertions.Nil(err, "failed to open file") {
							return
						}
						defer rd.Close()

						computedChecksum, err := ioutils.ChecksumSha256(ctx, rd)
						if !assertions.Nil(err, "failed to hash contents") {
							return
						}

						t.Logf("Checksum[%s]: %s", entry, computedChecksum)
						t.Run("ChecksumSha256", func(t *testing.T) {
							assertions := assert.New(t)

							ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
							defer cancel()
							remoteChecksum, err := gd.ChecksumSha256(ctx, entry.Location())
							if !assertions.Nil(err, "failed to calculate checksum") {
								return
							}

							t.Logf("Computed Checksum: %s", computedChecksum)
							t.Logf("Remote checksum: %s", remoteChecksum)
							if !assertions.Equal(remoteChecksum, computedChecksum, "checksums doesn't match") {
								return
							}
						})

						t.Run("Remote Checksum", func(t *testing.T) {
							assertions := assert.New(t)

							ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
							defer cancel()

							remoteChecksum, err := gd.ChecksumSha256(ctx, entry.Location())
							if !assertions.Nil(err, "failed to compute request checksum") {
								return
							}

							if !assertions.Equal(computedChecksum, remoteChecksum, "remote checksum doesn't match local checksum") {
								return
							}
						})
					})
				}

				assertions.NotZero(index, "no files found")
			})
		}
	})
}
