package filesystem

import (
	"context"
	"io"
	"iter"
	"log"
	"path"

	"github.com/pluto-org-co/fsio/googleutils/directory"
	"github.com/pluto-org-co/fsio/googleutils/drives"
	"github.com/pluto-org-co/fsio/googleutils/shareddrives"
	"golang.org/x/oauth2/jwt"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDrive struct {
	jwtLoader    func() (config *jwt.Config)
	otherUsers   bool
	sharedDrives bool
}

func NewGoogleDrive(conf GoogleDriveConfig) (g *GoogleDrive) {
	return &GoogleDrive{
		jwtLoader:    conf.JWTLoader,
		otherUsers:   conf.OtherUsers,
		sharedDrives: conf.SharedDrive,
	}
}

var _ Filesystem = (*GoogleDrive)(nil)

type GoogleDriveConfig struct {
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
}

func (g *GoogleDrive) currentUserFilename(filename string) (finalFileme string) {
	return path.Join("personal", "files", filename)
}

func (g *GoogleDrive) currentSharedDriveFilename(driveName, filename string) (finalFileme string) {
	return path.Join("drives", driveName, "files", filename)
}

func (g *GoogleDrive) userAccountDriveFilename(domain, username, filename string) (finalFilename string) {
	return path.Join("domains", domain, "users", username, "files", filename)
}

func (g *GoogleDrive) Files(ctx context.Context) (seq iter.Seq[string]) {
	baseConf := g.jwtLoader()
	baseClient := baseConf.Client(ctx)

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		log.Printf("failed to get drive service: %v", err)
		return func(yield func(string) bool) {}
	}

	adminSvc, err := admin.NewService(ctx, option.WithHTTPClient(baseClient))
	if err != nil {
		log.Printf("failed to get admin service: %v", err)
		return func(yield func(string) bool) {}
	}

	return func(yield func(string) bool) {
		// Start with the files owned by this account.
		for filename := range drives.SeqFiles(ctx, driveSvc) {
			if !yield(g.currentUserFilename(filename)) {
				return
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
	return
}

func (g *GoogleDrive) WriteFile(ctx context.Context, filePath string, src io.Reader) (filename string, err error) {
	return
}

func (g *GoogleDrive) RemoveAll(ctx context.Context, filePath string) (err error) {
	return
}
