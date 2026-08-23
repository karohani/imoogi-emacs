package main

import (
	"context"
	"os"

	"github.com/karohani/imoogi-emacs/internal/cli"
	"github.com/karohani/imoogi-emacs/internal/fetch"
	"github.com/karohani/imoogi-emacs/internal/setup"
)

var version = "dev"

func main() {
	workdir, err := os.Getwd()
	if err != nil {
		workdir = "."
	}
	os.Exit(cli.Run(os.Args[1:], cli.Config{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: version,
		Workdir: workdir,
		Hooks: cli.Hooks{
			Fetch: func(ctx context.Context, io cli.IO) error {
				return fetch.Run(ctx, fetch.Options{
					Workdir: workdir,
					Stdout:  io.Stdout,
					Stderr:  io.Stderr,
				})
			},
			Setup: func(ctx context.Context, io cli.IO) error {
				return setup.Run(ctx, setup.Options{
					Workdir: workdir,
					Stdout:  io.Stdout,
					Stderr:  io.Stderr,
				})
			},
		},
	}))
}
