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

package shareddrives_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"
	"testing"
	"time"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/shareddrives"
	"github.com/stretchr/testify/assert"
	admin "google.golang.org/api/admin/directory/v1"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func Test_Open(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		assertions := assert.New(t)

		conf := creds.NewConfiguration(t, googleutils.Scopes...)
		conf.Subject = creds.UserEmail()

		ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
		defer cancel()
		client := conf.Client(ctx)

		adminSvc, err := admin.NewService(ctx, option.WithHTTPClient(client))
		if !assertions.Nil(err, "failed to create service") {
			return
		}

		for domain := range directory.SeqDomains(ctx, adminSvc) {
			t.Logf("Domain: %s", domain.DomainName)

			t.Run(domain.DomainName, func(t *testing.T) {
				assertions := assert.New(t)
				var totalCount int
				for u := range directory.SeqUsers(ctx, adminSvc, domain.DomainName) {
					t.Run(u.PrimaryEmail, func(t *testing.T) {
						assertions := assert.New(t)

						conf := creds.NewConfiguration(t, googleutils.Scopes...)
						conf.Subject = u.PrimaryEmail

						ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
						defer cancel()
						client := conf.Client(ctx)

						driveSvc, err := gdrive.NewService(ctx, option.WithHTTPClient(client))
						if !assertions.Nil(err, "failed to create service") {
							return
						}

						for driveEntry := range shareddrives.SeqDrives(ctx, driveSvc) {
							t.Logf("Drive: %v", driveEntry.Name)
							t.Run(driveEntry.Name, func(t *testing.T) {
								ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
								defer cancel()

								var count int
								for location, file := range shareddrives.SeqFiles(ctx, driveSvc, driveEntry.Id) {
									t.Logf("[%d] File: %s - %v", count, location, file.Id)
									count++
									if count >= 5 {
										break
									}

									t.Run(path.Join(location...), func(t *testing.T) {
										assertions := assert.New(t)

										ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
										defer cancel()
										rc, err := shareddrives.Open(ctx, driveSvc, driveEntry.Id, location)
										if !assertions.Nil(err, "failed to open filename") {
											return
										}
										defer rc.Close()

										hash := sha256.New()
										_, err = io.Copy(hash, bufio.NewReader(rc))
										if !assertions.Nil(err, "failed to hash file") {
											return
										}

										checksum := hex.EncodeToString(hash.Sum(nil))
										t.Logf("Checksum[%s]: %s", location, checksum)
									})
								}
								totalCount += count
							})
							if totalCount >= 10 {
								break
							}
						}
					})
					if totalCount >= 10 {
						break
					}
				}
				if !assertions.NotZero(totalCount, "expecting at least one file") {
					return
				}
			})
		}
	})
}
