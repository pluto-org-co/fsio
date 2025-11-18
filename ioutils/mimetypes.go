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

package ioutils

var GoogleMimeTypes = []string{
	// Native Google Workspace Formats
	"application/vnd.google-apps.document",
	"application/vnd.google-apps.spreadsheet",
	"application/vnd.google-apps.presentation",

	// Other Google MimeTypes
	"application/vnd.google-apps.drawing",
	"application/vnd.google-apps.form",
}

var DocsLikeMimeTypes = []string{
	"application/vnd.google-apps.document",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document", // .docx
	"application/msword",                      // .doc
	"application/vnd.oasis.opendocument.text", // .odt
}

var OfficeMimeTypes = []string{
	// Microsoft Office Formats (OOXML)
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",   // .docx
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",         // .xlsx
	"application/vnd.openxmlformats-officedocument.presentationml.presentation", // .pptx

	// Other office MimeTypes
	"application/msword",            // .doc
	"application/vnd.ms-excel",      // .xls
	"application/vnd.ms-powerpoint", // .ppt
}

var OpenOfficeMimeTypes = []string{
	"application/vnd.oasis.opendocument.text",         // .odt
	"application/vnd.oasis.opendocument.spreadsheet",  // .ods
	"application/vnd.oasis.opendocument.presentation", // .odp
	"application/vnd.oasis.opendocument.graphics",     // .odg
	"application/vnd.oasis.opendocument.formula",      // .odf
}

var OfficeLikeMimeTypes = func() (mimetypes []string) {
	mimetypes = make([]string, 0, len(GoogleMimeTypes)+len(OfficeMimeTypes))

	mimetypes = append(mimetypes, GoogleMimeTypes...)
	mimetypes = append(mimetypes, OfficeMimeTypes...)
	mimetypes = append(mimetypes, OpenOfficeMimeTypes...)
	return mimetypes
}()
