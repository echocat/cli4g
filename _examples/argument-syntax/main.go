package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	File string   `argument:"file" default:"input.txt" description:"First file name."`
	More []string `argument:"more" description:"Additional file names."`
}

func main() {
	cli.NewApplication[options]().
		WithName("argument-syntax").
		WithDescription("Optional arguments, variadic arguments, and the -- terminator.").
		WithHandler(showArguments).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showArguments(_ context.Context, opts options) error {
	_, err := fmt.Printf("file=%s more=%v\n", opts.File, opts.More)
	return err
}
