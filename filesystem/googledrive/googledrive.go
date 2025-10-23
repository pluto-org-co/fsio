package googledrive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"slices"

	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/drives"
	"github.com/pluto-org-co/fsio/googleutils/shareddrives"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDrive struct {
	jwtLoader      func() (config *jwt.Config)
	otherUsers     bool
	sharedDrives   bool
	currentAccount bool
}

func New(conf Config) (g *GoogleDrive) {
	return &GoogleDrive{
		jwtLoader:      conf.JWTLoader,
		otherUsers:     conf.OtherUsers,
		sharedDrives:   conf.SharedDrive,
		currentAccount: conf.CurrentAccount,
	}
}

var _ filesystem.Filesystem = (*GoogleDrive)(nil)

type Config struct {
	// Constructs the config loader used for preparing the service.
	// The mandatory permission for Service account is https://www.googleapis.com/auth/drive
	JWTLoader func() (config *jwt.Config)
	// Query from other users drive. Depends on AdminService for listing the users
	// This may require the following permissions
	// https://www.googleapis.com/auth/admin.directory.domain.readonly
	// https://www.googleapis.com/auth/admin.directory.user.readonly
	OtherUsers bool
	// Include shared drives with the current user.
	SharedDrive bool
	// Handle files of the current account
	CurrentAccount bool
}

func (g *GoogleDrive) currentUserFilename(location []string) (finalLocation []string) {
	finalLocation = make([]string, 2+len(location))
	finalLocation = append(finalLocation, "personal", "files")
	finalLocation = append(finalLocation, location...)
	return finalLocation
}

func (g *GoogleDrive) filenameIsCurrentUser(location []string) (ok bool, realLocation []string) {
	if len(location) >= 2 && location[0] == "personal" && location[1] == "files" {
		return true, slices.Clone(location[2:])
	}
	return false, nil
}

func (g *GoogleDrive) currentSharedDriveFilename(driveName string, location []string) (finalLocation []string) {
	finalLocation = make([]string, 3+len(location))
	finalLocation = append(finalLocation, "drives", driveName, "files")
	finalLocation = append(finalLocation, location...)
	return finalLocation
}

func (g *GoogleDrive) filenameIsCurrentSharedDrives(location []string) (ok bool, drivename string, realLocation []string) {
	if len(location) >= 3 && location[0] == "drives" && location[2] == "files" {
		return true, location[1], slices.Clone(location[3:])
	}
	return false, "", nil
}

func (g *GoogleDrive) userAccountDriveFilename(domain, username string, location []string) (finalLocation []string) {
	finalLocation = make([]string, 5+len(location))
	finalLocation = append(finalLocation, "domain", domain, "users", username, "files")
	finalLocation = append(finalLocation, location...)
	return finalLocation
}

func (g *GoogleDrive) filenameIsUserAccountDrive(location []string) (ok bool, domain, username string, realLocation []string) {
	if len(location) >= 5 && location[0] == "domains" && location[2] == "users" && location[4] == "files" {
		return true, location[1], location[3], slices.Clone(location[5:])
	}
	return false, "", "", nil
}

func (g *GoogleDrive) Checksum(ctx context.Context, location []string) (checksum string, err error) {
	baseConf := g.jwtLoader()
	baseClient := baseConf.Client(ctx)

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		return "", fmt.Errorf("failed to prepare drive service: %w", err)
	}

	if g.currentAccount {
		ok, filename := g.filenameIsCurrentUser(location)
		if ok {
			checksum, err := drives.Checksum(ctx, driveSvc, filename)
			if err != nil {
				return "", fmt.Errorf("failed to compute current user checksum: %w", err)
			}
			return checksum, nil
		}
	}

	if g.sharedDrives {
		ok, driveName, filename := g.filenameIsCurrentSharedDrives(location)
		if ok {
			var driveId string
			for drive := range shareddrives.SeqDrives(ctx, driveSvc) {
				if drive.Name == driveName {
					driveId = drive.Id
					break
				}
			}

			if driveId == "" {
				return "", fmt.Errorf("drive not found by name: %s", driveName)
			}

			checksum, err := shareddrives.Checksum(ctx, driveSvc, driveId, filename)
			if err != nil {
				return "", fmt.Errorf("failed to compute shared drive checksum: %w", err)
			}
			return checksum, nil
		}
	}

	if g.otherUsers {
		ok, _, username, filename := g.filenameIsUserAccountDrive(location)
		if ok {
			userConf := g.jwtLoader()
			userConf.Subject = username

			userSvc, err := drive.NewService(ctx, option.WithHTTPClient(userConf.Client(ctx)))
			if err != nil {
				return "", fmt.Errorf("failed to prepare client for user: %w", err)
			}

			checksum, err := drives.Checksum(ctx, userSvc, filename)
			if err != nil {
				return "", fmt.Errorf("failed to compute checksum for user file: %w", err)
			}
			return checksum, nil
		}
	}

	return "", fmt.Errorf("file not found: %v", location)
}

func (g *GoogleDrive) Files(ctx context.Context) (seq iter.Seq[[]string]) {
	baseConf := g.jwtLoader()
	baseClient := baseConf.Client(ctx)

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		log.Printf("failed to get drive service: %v", err)
		return func(yield func([]string) bool) {}
	}

	adminSvc, _ := admin.NewService(ctx, option.WithHTTPClient(baseClient))

	return func(yield func([]string) bool) {
		// Start with the files owned by this account.
		if g.currentAccount {
			for filename := range drives.SeqFiles(ctx, driveSvc) {
				if !yield(g.currentUserFilename(filename)) {
					return
				}
			}
		}

		if g.sharedDrives {
			for drive := range shareddrives.SeqDrives(ctx, driveSvc) {
				for filename := range shareddrives.SeqFiles(ctx, driveSvc, drive.Id) {
					if !yield(g.currentSharedDriveFilename(drive.Name, filename)) {
						return
					}
				}
			}
		}

		if g.otherUsers && adminSvc != nil {
			for domain := range directory.SeqDomains(ctx, adminSvc) {
				for user := range directory.SeqUsers(ctx, adminSvc, domain.DomainName) {
					userConf := g.jwtLoader()
					userConf.Subject = user.PrimaryEmail

					userSvc, err := drive.NewService(ctx, option.WithHTTPClient(userConf.Client(ctx)))
					if err != nil {
						log.Printf("failed to load user configuration: %v", err)
						return
					}
					for filename := range drives.SeqFiles(ctx, userSvc) {
						if !yield(g.userAccountDriveFilename(domain.DomainName, user.PrimaryEmail, filename)) {
							return
						}
					}
				}
			}
		}
	}
}

func (g *GoogleDrive) Open(ctx context.Context, location []string) (rc io.ReadCloser, err error) {
	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(g.jwtLoader().Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	if g.currentAccount {
		ok, filename := g.filenameIsCurrentUser(location)
		if ok {
			rc, err := drives.Open(ctx, driveSvc, filename)
			if err != nil {
				return nil, fmt.Errorf("failed to open current user file: %w", err)
			}
			return rc, nil
		}
	}

	if g.sharedDrives {
		ok, drivename, filename := g.filenameIsCurrentSharedDrives(location)
		if ok {
			var driveId string
			for driveEntry := range shareddrives.SeqDrives(ctx, driveSvc) {
				if driveEntry.Name == drivename {
					driveId = driveEntry.Id
					break
				}
			}
			if driveId == "" {
				return nil, fmt.Errorf("failed to find drive by its name: %w", err)
			}

			rc, err := shareddrives.Open(ctx, driveSvc, driveId, filename)
			if err != nil {
				return nil, fmt.Errorf("failed to open drive file: %w", err)
			}
			return rc, nil
		}
	}

	if g.otherUsers {
		ok, _, username, filename := g.filenameIsUserAccountDrive(location)
		if ok {
			baseConf := g.jwtLoader()
			baseConf.Subject = username

			driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseConf.Client(ctx)))
			if err != nil {
				return nil, fmt.Errorf("failed to create drive service: %w", err)
			}

			rc, err := drives.Open(ctx, driveSvc, filename)
			if err != nil {
				return nil, fmt.Errorf("failed to open user file: %s: %w", username, err)
			}
			return rc, nil
		}
	}
	return nil, errors.New("file not found")
}

func (g *GoogleDrive) WriteFile(ctx context.Context, location []string, src io.Reader) (finalLocation []string, err error) {
	return nil, errors.New("operation not supported")
}

func (g *GoogleDrive) RemoveAll(ctx context.Context, location []string) (err error) {
	return errors.New("operation not supported")
}
