package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/driveutils"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const PersonalDrive = "personal"

const (
	CredentialsFlag = "creds"
	SubjectFlag     = "subject"
	RoleFlag        = "role"
	TypeFlag        = "type"
)

const (
	FilePathArg   = "FILE_PATH"
	TargetUserArg = "TARGET_USER"
	DriveNameFlag = "drive"
)

func ConfigFromFile(file, subject string) (conf *jwt.Config, err error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", file, err)
	}

	conf, err = google.JWTConfigFromJSON(contents, googleutils.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from file: %w", err)
	}
	conf.Subject = subject
	return conf, nil
}

var ShareCmd = cli.Command{
	Name: "share",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     CredentialsFlag,
			Usage:    "Service account JSON credentials file",
			Required: true,
			OnlyOnce: true,
		},
		&cli.StringFlag{
			Name:     SubjectFlag,
			Usage:    "Subject of the account",
			Required: true,
			OnlyOnce: true,
		},
		&cli.StringFlag{
			Name:     DriveNameFlag,
			Usage:    "Name of the drive",
			Value:    PersonalDrive,
			OnlyOnce: true,
		},
		&cli.StringFlag{
			Name:     RoleFlag,
			Usage:    "Role of the permission to apply",
			Required: true,
			OnlyOnce: true,
			Validator: func(s string) error {
				var options = []string{"owner", "organizer", "fileOrganizer", "writer", "commenter", "reader"}
				if slices.Contains(options, s) {
					return nil
				}
				return fmt.Errorf("unknown role: %s: expecting: %s", s, strings.Join(options, ", "))
			},
		},
		&cli.StringFlag{
			Name:     TypeFlag,
			Usage:    "Type of the permission to apply",
			Required: true,
			OnlyOnce: true,
			Validator: func(s string) error {
				var options = []string{"group", "user", "domain", "anyone"}
				if slices.Contains(options, s) {
					return nil
				}
				return fmt.Errorf("unknown type: %s: expecting: %s", s, strings.Join(options, ", "))
			},
		},
	},
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      FilePathArg,
			UsageText: "File path of the remote file",
		},
		&cli.StringArg{
			Name:      TargetUserArg,
			UsageText: "Target user email",
		},
	},
	ArgsUsage: "FILE_PATH TARGET_USER",
	Action: func(ctx context.Context, c *cli.Command) (err error) {
		conf, err := ConfigFromFile(c.String(CredentialsFlag), c.String(SubjectFlag))
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		httpClient := conf.Client(ctx)

		role := c.String(RoleFlag)
		permType := c.String(TypeFlag)
		driveName := c.String(DriveNameFlag)
		file := c.StringArg(FilePathArg)
		targetUser := c.StringArg(TargetUserArg)

		switch {
		case file == "":
			return fmt.Errorf("%s not specified", FilePathArg)
		case targetUser == "":
			return fmt.Errorf("%s not specified", TargetUserArg)
		}

		driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
		if err != nil {
			return fmt.Errorf("failed to create drive service: %w", err)
		}

		var fileId string
		var perm = drive.Permission{
			EmailAddress: targetUser,
			Role:         role,
			Type:         permType,
		}
		switch driveName {
		case PersonalDrive:
			location := strings.Split(file, "/")
			reference, err := driveutils.FindFileByPath(ctx, location, "root", func() *drive.FilesListCall {
				return driveSvc.Files.List().Corpora("user")
			})
			if err != nil {
				return fmt.Errorf("failed to retrieve file by its location: %w", err)
			}

			fileId = reference.Id
		default:
		}

		_, err = driveSvc.Permissions.
			Create(fileId, &perm).
			Context(ctx).
			SupportsAllDrives(true).
			SupportsTeamDrives(true).
			Do()
		if err != nil {
			return fmt.Errorf("failed to create permission: %w", err)
		}
		return nil
	},
}
