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
