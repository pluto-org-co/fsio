package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/drives"
	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"github.com/pluto-org-co/fsio/googleutils/shareddrives"
	"github.com/pluto-org-co/fsio/ioutils"
	"github.com/urfave/cli/v3"
	"golang.org/x/exp/slog"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

const ConfigFlag = "config"

func ClientFromSubject(ctx context.Context, config *jwt.Config, subject string) (client *http.Client) {
	config.Subject = subject
	client = config.Client(ctx)
	client.Transport = ioutils.NewRetryTransport(client.Transport, 100, time.Minute)
	return client
}

func UnshareFile(workerPool chan struct{}, wg *sync.WaitGroup, logger *slog.Logger, driveSvc *drive.Service, ctx context.Context, user *admin.User, file *drive.File) (err error) {
	<-workerPool
	wg.Go(func() {
		defer func() { workerPool <- struct{}{} }()
		logger := logger.With("id", file.Id)
		logger.Debug("Found shared file")
		err = driveutils.UnshareAll(ctx, driveSvc, file.Id)
		if err != nil {
			logger.Error("failed to unshare file", "error-msg", err)
			return
		}
		logger.Debug("Removed permissions")
	})
	return nil
}

func UnshareFiles(workerPool chan struct{}, wg *sync.WaitGroup, logger *slog.Logger, driveSvc *drive.Service, ctx context.Context, user *admin.User, files []*drive.File) (err error) {
	for _, file := range files {
		logger := logger.With("filename", file.Name)

		err = UnshareFile(workerPool, wg, logger, driveSvc, ctx, user, file)
		if err != nil {
			return fmt.Errorf("failed to unshare file: %s: %w", file.Name, err)
		}
	}
	return nil
}

var Unshare = cli.Command{
	Name: "unshare",
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

		var cfg Config
		err = yaml.Unmarshal(contents, &cfg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal contents: %w", err)
		}

		accountFile, err := os.ReadFile(cfg.Drive.AccountFile)
		if err != nil {
			return fmt.Errorf("failed to get account file: %w", err)
		}

		config, err := google.JWTConfigFromJSON(accountFile, googleutils.Scopes...)
		if err != nil {
			return fmt.Errorf("failed to prepare jwt configuration: %w", err)
		}

		adminSvc, err := admin.NewService(ctx, option.WithHTTPClient(ClientFromSubject(ctx, config, cfg.Drive.Subject)))
		if err != nil {
			return fmt.Errorf("failed to get admin service: %w", err)
		}

		const poolSize = 10
		var workerPool = make(chan struct{}, poolSize)
		for range poolSize {
			workerPool <- struct{}{}
		}
		var wg sync.WaitGroup
		wg.Wait()

		logger := slog.With("subject", cfg.Drive.Subject)

		rootDriveService, err := drive.NewService(ctx, option.WithHTTPClient(ClientFromSubject(ctx, config, cfg.Drive.Subject)))
		if err != nil {
			return fmt.Errorf("failed create root drive service: %w", err)
		}

		{
			logger := logger.With("mode", "drives")
			logger.Info("Processing")
			for driveEntry := range shareddrives.SeqDrives(ctx, rootDriveService) {
				logger := logger.With("drive", driveEntry.Name)

				logger.Info("Processing directories")
				var driveAccessedDirectories = make([]*drive.File, 0, 100)
				err = rootDriveService.Files.
					List().
					SupportsAllDrives(true).
					IncludeItemsFromAllDrives(true).
					IncludeTeamDriveItems(true).
					Corpora("drive").
					DriveId(driveEntry.Id).
					PageSize(1000).
					Q("mimeType = 'application/vnd.google-apps.folder'").
					Fields("nextPageToken,files(id,name)").
					Pages(ctx, func(fl *drive.FileList) (err error) {
						driveAccessedDirectories = append(driveAccessedDirectories, fl.Files...)
						return nil
					})
				if err != nil {
					return fmt.Errorf("first target directories: %w", err)
				}

				logger.Info("Found directories", "count", len(driveAccessedDirectories))
				err = UnshareFiles(workerPool, &wg, logger, rootDriveService, ctx, nil, driveAccessedDirectories)
				if err != nil {
					return fmt.Errorf("failed to unshare drive directories: %w", err)
				}

				logger.Info("Processing files")
				for _, file := range shareddrives.SeqFiles(ctx, rootDriveService, driveEntry.Id) {
					logger := logger.With("file", file.Name)

					err = UnshareFile(workerPool, &wg, logger, rootDriveService, ctx, nil, file)
					if err != nil {
						return fmt.Errorf("failed to process file: %w", err)
					}
				}
			}
		}

		{
			logger := logger.With("mode", "users")
			logger.Info("Processing")
			for domain := range directory.SeqDomains(ctx, adminSvc) {
				logger := slog.With("domain", domain.DomainName)

				for user := range directory.SeqUsers(ctx, adminSvc, domain.DomainName) {
					logger := logger.With("user", user.PrimaryEmail)
					userDriveSvc, err := drive.NewService(ctx, option.WithHTTPClient(ClientFromSubject(ctx, config, user.PrimaryEmail)))
					if err != nil {
						return fmt.Errorf("failed to create drive service: %w", err)
					}

					logger.Info("Processing directories")
					var userAccessedDirectories []*drive.File
					err = userDriveSvc.Files.
						List().
						Corpora("user").
						Q("mimeType = 'application/vnd.google-apps.folder'").
						PageSize(1000).
						Fields("nextPageToken,files(id,name)").
						Pages(ctx, func(fl *drive.FileList) (err error) {
							userAccessedDirectories = append(userAccessedDirectories, fl.Files...)
							return nil
						})
					if err != nil {
						return fmt.Errorf("first target directories: %w", err)
					}

					err = UnshareFiles(workerPool, &wg, logger, userDriveSvc, ctx, user, userAccessedDirectories)
					if err != nil {
						return fmt.Errorf("failed to unshare files: %w", err)
					}

					logger.Info("Processing Files")
					for _, file := range drives.SeqFiles(ctx, userDriveSvc) {
						logger := logger.With("file", file.Name)

						err = UnshareFile(workerPool, &wg, logger, userDriveSvc, ctx, nil, file)
						if err != nil {
							return fmt.Errorf("failed to process file: %w", err)
						}
					}
				}
			}
		}
		return nil
	},
}
