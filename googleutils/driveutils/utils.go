package driveutils

import "strings"

func RemoveSlashFromPart(part string) (final string) {
	return strings.ReplaceAll(part, "/", "%2F")
}

func AddSlashToFilename(filename string) (final string) {
	return strings.ReplaceAll(filename, "%2F", "/")
}
