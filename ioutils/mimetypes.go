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

var OfficeLikeMimeTypes = func() (mimetypes []string) {
	mimetypes = make([]string, 0, len(GoogleMimeTypes)+len(OfficeMimeTypes))

	mimetypes = append(mimetypes, GoogleMimeTypes...)
	mimetypes = append(mimetypes, OfficeMimeTypes...)
	return mimetypes
}()
