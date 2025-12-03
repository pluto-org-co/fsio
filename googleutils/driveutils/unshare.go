package driveutils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"google.golang.org/api/drive/v3"
)

// Remove all file access to a document
func Unshare(ctx context.Context, svc *drive.Service, fileId string, permissions []*drive.Permission) (err error) {
	if len(permissions) == 0 {
		return errors.New("no permission id provided")
	}
	// Remove any file access to a document
	for _, permission := range permissions {
		err = svc.Permissions.
			Delete(fileId, permission.Id).
			Context(ctx).
			SupportsAllDrives(true).
			SupportsTeamDrives(true).
			EnforceExpansiveAccess(true).
			Do()
		if err != nil {
			slog.Debug("Failed to remove permission",
				"file-id", fileId,
				"error-msg", err,
			)
		}
	}
	return nil
}

// Remove all file access to a document
func UnshareAll(ctx context.Context, svc *drive.Service, fileId string) (err error) {
	var permissions []*drive.Permission
	err = svc.Permissions.
		List(fileId).
		Fields("nextPageToken,permissions(id)").
		PageSize(100).
		SupportsAllDrives(true).
		SupportsTeamDrives(true).
		Pages(ctx, func(pl *drive.PermissionList) (err error) {
			permissions = append(permissions, pl.Permissions...)
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to list file permissions: %w", err)
	}

	// Remove any file access to a document
	err = Unshare(ctx, svc, fileId, permissions)
	if err != nil {
		return fmt.Errorf("failed to unshare file: %w", err)
	}
	return nil
}
