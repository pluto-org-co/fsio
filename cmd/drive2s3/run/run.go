package run

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pluto-org-co/fsio/filesystem"

	"github.com/pluto-org-co/fsio/cmd/drive2s3/config"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

var ConfigFlag = "config"

var RunCommand = &cli.Command{
	Name:        "run",
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

		var cfg config.Config
		err = yaml.Unmarshal(contents, &cfg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal contents: %w", err)
		}

		log.Println("Preparing S3 FS")
		s3Fs, err := cfg.S3Fs(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare s3 fs: %w", err)
		}

		log.Println("Preparing Drive FS")
		driveFs, err := cfg.DriveFs(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare drive fs: %w", err)
		}

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for {
			log.Println("Syncing")
			err = filesystem.SyncWorkers(cfg.Workers, ctx, s3Fs, driveFs)
			if err != nil {
				return fmt.Errorf("failed to sync: %w", err)
			}

			log.Println("Waiting next sync")
			<-ticker.C
		}
	},
}
