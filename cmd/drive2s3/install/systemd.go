package install

import _ "embed"

//go:embed systemd.service
var serviceFilContents []byte
