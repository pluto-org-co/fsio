package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/filesystem/s3"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v2"
	"gopkg.in/yaml.v3"
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
		Workers int   `yaml:"workers"`
		Drive   Drive `yaml:"drive"`
		S3      S3    `yaml:"s3"`
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

	_, err = google.JWTConfigFromJSON(accountFile,
		admin.AdminDirectoryUserReadonlyScope,
		admin.AdminDirectoryDomainReadonlyScope,
		drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare jwt configuration: %w", err)
	}

	gd := googledrive.New(googledrive.Config{
		JWTLoader: func() (config *jwt.Config) {
			config, _ = google.JWTConfigFromJSON(accountFile,
				admin.AdminDirectoryUserReadonlyScope,
				admin.AdminDirectoryDomainReadonlyScope,
				drive.DriveScope)
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

var ConfigFlag = "config"

var app = cli.Command{
	Name:        "drive2s3",
	Description: "sync the contents of a google drive with a s3 bucket",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  ConfigFlag,
			Value: "config.yaml",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) (err error) {
		contents, err := os.ReadFile(c.String(ConfigFlag))
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var config Config
		err = yaml.Unmarshal(contents, &config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal contents: %w", err)
		}

		log.Println("Preparing S3 FS")
		s3Fs, err := config.S3Fs(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare s3 fs: %w", err)
		}

		log.Println("Preparing Drive FS")
		driveFs, err := config.DriveFs(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare drive fs: %w", err)
		}

		log.Println("Syncing")
		err = filesystem.SyncWorkers(config.Workers, ctx, s3Fs, driveFs)
		if err != nil {
			return fmt.Errorf("failed to sync: %w", err)
		}

		return nil
	},
}

func main() {
	ctx := context.TODO()

	err := app.Run(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
