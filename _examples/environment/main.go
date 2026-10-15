package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Profile string `flag:"profile" short:"p" env:"CLI4G_PROFILE" default:"development" description:"Configuration profile."`
	Port    uint16 `flag:"port" env:"CLI4G_PORT" default:"8080" description:"Port to use."`
}

func main() {
	cli.NewApplication[options]().
		WithName("environment").
		WithDescription("CLI values override environment variables and defaults.").
		WithHandler(showOptions).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showOptions(_ context.Context, opts options) error {
	_, err := fmt.Printf("profile=%s port=%d\n", opts.Profile, opts.Port)
	return err
}
