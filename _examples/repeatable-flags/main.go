package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Tags []string `flag:"tag" short:"t" description:"Add a tag; may be repeated."`
}

func main() {
	cli.NewApplication[options]().
		WithName("repeatable-flags").
		WithDescription("Collect repeated flag values in a slice.").
		WithHandler(showTags).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showTags(_ context.Context, opts options) error {
	_, err := fmt.Printf("tags=%v\n", opts.Tags)
	return err
}
