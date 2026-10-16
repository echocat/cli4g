package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Tags []string `flag:"tag" short:"t" description:"Add a tag; may be repeated."`
}

func main() {
	cli.NewApplication[options]("repeatable-flags", "Collect repeated flag values in a slice.").
		WithHandler(showTags).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showTags(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "tags=%v\n", opts.Tags)
}
