package googledrive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"path"
	"strings"

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

func (g *GoogleDrive) currentUserFilename(filename string) (finalFileme string) {
	return path.Join("personal", "files", filename)
}

func (g *GoogleDrive) filenameIsCurrentUser(filename string) (ok bool, realFilename string) {
	if path.IsAbs(filename) {
		filename = filename[1:]
	}

	parts := strings.Split(filename, "/")

	if len(parts) >= 2 && parts[0] == "personal" && parts[1] == "files" {
		return true, path.Join(parts[2:]...)
	}
	return false, ""
}

func (g *GoogleDrive) currentSharedDriveFilename(driveName, filename string) (finalFileme string) {
	return path.Join("drives", driveName, "files", filename)
}

func (g *GoogleDrive) filenameIsCurrentSharedDrives(filename string) (ok bool, drivename, realFilename string) {
	if path.IsAbs(filename) {
		filename = filename[1:]
	}

	parts := strings.Split(filename, "/")

	if len(parts) >= 3 && parts[0] == "drives" && parts[2] == "files" {
		return true, parts[1], path.Join(parts[3:]...)
	}
	return false, "", ""
}

func (g *GoogleDrive) userAccountDriveFilename(domain, username, filename string) (finalFilename string) {
	return path.Join("domains", domain, "users", username, "files", filename)
}

func (g *GoogleDrive) filenameIsUserAccountDrive(filename string) (ok bool, domain, username, realFilename string) {
	if path.IsAbs(filename) {
		filename = filename[1:]
	}

	parts := strings.Split(filename, "/")

	if len(parts) >= 5 && parts[0] == "domains" && parts[2] == "users" && parts[4] == "files" {
		return true, parts[1], parts[3], path.Join(parts[5:]...)
	}
	return false, "", "", ""
}

func (g *GoogleDrive) Checksum(ctx context.Context, filePath string) (checksum string, err error) {
	baseConf := g.jwtLoader()
	baseClient := baseConf.Client(ctx)

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		return "", fmt.Errorf("failed to prepare drive service: %w", err)
	}

	if g.currentAccount {
		ok, filename := g.filenameIsCurrentUser(filePath)
		if ok {
			checksum, err := drives.Checksum(ctx, driveSvc, filename)
			if err != nil {
				return "", fmt.Errorf("failed to compute current user checksum: %w", err)
			}
			return checksum, nil
		}
	}

	if g.sharedDrives {
		ok, driveName, filename := g.filenameIsCurrentSharedDrives(filePath)
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
		ok, _, username, filename := g.filenameIsUserAccountDrive(filePath)
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

	return "", fmt.Errorf("file not found: %s", filePath)
}

func (g *GoogleDrive) Files(ctx context.Context) (seq iter.Seq[string]) {
	baseConf := g.jwtLoader()
	baseClient := baseConf.Client(ctx)

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		log.Printf("failed to get drive service: %v", err)
		return func(yield func(string) bool) {}
	}

	adminSvc, _ := admin.NewService(ctx, option.WithHTTPClient(baseClient))

	return func(yield func(string) bool) {
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

func (g *GoogleDrive) Open(ctx context.Context, filePath string) (rc io.ReadCloser, err error) {
	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(g.jwtLoader().Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	if g.currentAccount {
		ok, filename := g.filenameIsCurrentUser(filePath)
		if ok {
			rc, err := drives.Open(ctx, driveSvc, filename)
			if err != nil {
				return nil, fmt.Errorf("failed to open current user file: %w", err)
			}
			return rc, nil
		}
	}

	if g.sharedDrives {
		ok, drivename, filename := g.filenameIsCurrentSharedDrives(filePath)
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
		ok, _, username, filename := g.filenameIsUserAccountDrive(filePath)
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

func (g *GoogleDrive) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	return "", errors.New("operation not supported")
}

func (g *GoogleDrive) RemoveAll(ctx context.Context, filePath string) (err error) {
	return errors.New("operation not supported")
}
