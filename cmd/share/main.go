package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	err := ShareCmd.Run(context.TODO(), os.Args)
	if err != nil {
		slog.Error("failed to run", "error-msg", err)
	}
}
