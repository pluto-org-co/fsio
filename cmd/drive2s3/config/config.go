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

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/filesystem/s3"
	"github.com/pluto-org-co/fsio/googleutils"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

type (
	Drive struct {
		AccountFile    string `yaml:"account-file"`
		Subject        string `yaml:"subject"`
		CurrentAccount bool   `yaml:"current-account"`
		SharedDrive    bool   `yaml:"shared-drive"`
		OtherUsers     bool   `yaml:"other-users"`
	}
	S3 struct {
		Endpoint     string        `yaml:"endpoint"`
		ClientId     string        `yaml:"client-id"`
		ClientSecret string        `yaml:"client-secret"`
		Bucket       string        `yaml:"bucket"`
		CacheExpiry  time.Duration `yaml:"cache-expiry"`
	}
	Config struct {
		Workers  int           `yaml:"workers"`
		Interval time.Duration `yaml:"interval"`
		Drive    Drive         `yaml:"drive"`
		S3       S3            `yaml:"s3"`
	}
)

func (c *Config) S3Fs(ctx context.Context) (fs *s3.S3, err error) {
	client, err := minio.New(
		c.S3.Endpoint,
		&minio.Options{
			Creds:           credentials.NewStaticV4(c.S3.ClientId, c.S3.ClientSecret, ""),
			TrailingHeaders: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare client: %w", err)
	}

	_, err = client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	fs = s3.New(client, c.S3.Bucket, c.S3.CacheExpiry)
	return fs, nil
}

func (c *Config) DriveFs(ctx context.Context) (fs *googledrive.GoogleDrive, err error) {
	accountFile, err := os.ReadFile(c.Drive.AccountFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get account file: %w", err)
	}

	_, err = google.JWTConfigFromJSON(accountFile, googleutils.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare jwt configuration: %w", err)
	}

	gd := googledrive.New(googledrive.Config{
		JWTLoader: func() (config *jwt.Config) {
			config, _ = google.JWTConfigFromJSON(accountFile, googleutils.Scopes...)
			config.Subject = c.Drive.Subject
			return config
		},
		CurrentAccount: c.Drive.CurrentAccount,
		SharedDrive:    c.Drive.SharedDrive,
		OtherUsers:     c.Drive.OtherUsers,
	})

	var found bool
	for range gd.Files(ctx) {
		found = true
		break
	}

	if !found {
		return nil, errors.New("failed to test google drive fs")
	}
	return gd, nil
}

var Example = Config{
	Workers:  100,
	Interval: 24 * time.Hour,
	Drive: Drive{
		AccountFile:    "/path/to/redacted/svc-account.json",
		Subject:        "[REDACTED_ADMIN_EMAIL]",
		CurrentAccount: true,
		SharedDrive:    true,
		OtherUsers:     true,
	},
	S3: S3{
		Bucket:       "bucket-name",
		ClientId:     "[REDACTED_CLIENT_ID]",
		ClientSecret: "[REDACTED_CLIENT_SECRET]",
		Endpoint:     "[REDACTED_PRIVATE_ENDPOINT]",
		CacheExpiry:  time.Minute,
	},
}
