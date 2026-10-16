package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	File string   `argument:"file" default:"input.txt" description:"First file name."`
	More []string `argument:"more" description:"Additional file names."`
}

func main() {
	cli.NewApplication[options]("argument-syntax", "Optional arguments, variadic arguments, and the -- terminator.").
		WithHandler(showArguments).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showArguments(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "file=%s more=%v\n", opts.File, opts.More)
}
