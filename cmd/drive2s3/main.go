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

package main

import (
	"context"
	"log"
	"os"

	"github.com/pluto-org-co/fsio/cmd/drive2s3/install"
	"github.com/pluto-org-co/fsio/cmd/drive2s3/run"
	"github.com/urfave/cli/v3"
)

var Drive2S3 = cli.Command{
	Name: "drive2s3",
	Commands: []*cli.Command{
		run.RunCommand,
		install.InstallCommand,
	},
}

func main() {
	ctx := context.TODO()

	err := Drive2S3.Run(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
