package testsuite

import (
	"strings"

	"github.com/pluto-org-co/fsio/random"
)

func GenerateFilename(nParts int) (filename string) {
	if nParts == 0 {
		nParts = 1
	}

	var parts = make([]string, 0, nParts)
	for range nParts {
		parts = append(parts, random.InsecureString(5))
	}
	return strings.Join(parts, "/")
}

func GenerateFilenames(n int) (files []string) {
	files = make([]string, 0, n)
	for range n {
		files = append(files, GenerateFilename(random.InsecureInt(5)))
	}

	return files
}
