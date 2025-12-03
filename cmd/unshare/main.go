package main

import (
	"context"
	"log"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	ctx := context.TODO()
	err := Unshare.Run(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
