package driveutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/pluto-org-co/fsio/ioutils"
	"google.golang.org/api/drive/v3"
)

func Open(ctx context.Context, svc *drive.Service, mimeType, fileId string) (rd io.ReadCloser, err error) {
	if mimeType == "application/vnd.google-apps.folder" {
		return nil, errors.New("can't download folder")
	}

	var res *http.Response
	switch mimeType {
	case "application/vnd.google-apps.document": // Google Docs
		res, err = svc.Files.
			Export(fileId, "application/vnd.openxmlformats-officedocument.wordprocessingml.document").
			Context(ctx).
			Download()
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		defer res.Body.Close()
	case "application/vnd.google-apps.spreadsheet": // Google Sheets
		res, err = svc.Files.
			Export(fileId, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet").
			Context(ctx).
			Download()
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		defer res.Body.Close()
	case "application/vnd.google-apps.presentation": // Google Slides
		res, err = svc.Files.
			Export(fileId, "application/vnd.openxmlformats-officedocument.presentationml.presentation").
			Context(ctx).
			Download()
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		defer res.Body.Close()
	default:
		res, err = svc.Files.
			Get(fileId).
			Context(ctx).
			Download()
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		defer res.Body.Close()
	}

	file, err := ioutils.ReaderToTempFile(ctx, res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to store in temporary file: %w", err)
	}
	return file, nil
}
