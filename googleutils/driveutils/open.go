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

package driveutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"unsafe"

	"github.com/pluto-org-co/fsio/ioutils"
	"google.golang.org/api/drive/v3"
)

type ClientExtractor struct {
	Client *http.Client
}

var enumMimetypesOnce sync.Once
var exportMimetypes = map[string][]string{}

func Open(ctx context.Context, svc *drive.Service, mimeType, fileId string) (rd io.ReadCloser, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to open: %s: mimetype: %s: %w", fileId, mimeType, err)
		}
	}()
	enumMimetypesOnce.Do(func() {
		about, err := svc.About.Get().Fields("exportFormats").Context(ctx).Do()
		if err != nil {
			log.Printf("failed to query export mimetypes: %v", err)
			return
		}

		maps.Copy(exportMimetypes, about.ExportFormats)
	})
	if mimeType == "application/vnd.google-apps.folder" {
		return nil, errors.New("can't download folder")
	}

	var res *http.Response

	if export, found := exportMimetypes[mimeType]; found {
		fileInfo, err := svc.Files.
			Get(fileId).
			Fields("exportLinks").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}

		exportMimeIdx := slices.IndexFunc(export, func(targetExport string) bool { return strings.Contains(targetExport, "openxmlformats") })
		if exportMimeIdx == -1 {
			exportMimeIdx = 0
		}

		exportMime := export[exportMimeIdx]
		exportLink, found := fileInfo.ExportLinks[exportMime]
		if !found {
			res, err = svc.Files.
				Export(fileId, export[0]).
				Context(ctx).
				Download()
			if err != nil {
				return nil, fmt.Errorf("failed to export file: %w", err)
			}
			defer res.Body.Close()
		} else {
			client := (*ClientExtractor)(unsafe.Pointer(svc)).Client
			if client == nil {
				client = http.DefaultClient
			}

			res, err = client.Get(exportLink)
			if err != nil {
				return nil, fmt.Errorf("failed to use export link: %w", err)
			}
			defer res.Body.Close()
		}
	} else if mimeType == "application/vnd.google-apps.shortcut" {
		fileInfo, err := svc.Files.
			Get(fileId).
			Fields("shortcutDetails").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file information for shortcuts: %w", err)
		}
		if fileInfo.ShortcutDetails != nil && fileInfo.ShortcutDetails.TargetId != "" && fileInfo.ShortcutDetails.TargetMimeType != "application/vnd.google-apps.folder" {
			return Open(ctx, svc, fileInfo.ShortcutDetails.TargetMimeType, fileInfo.ShortcutDetails.TargetId)
		}
		return nil, errors.New("invalid shortcut")
	} else {
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
