package main

import (
	"context"
	"io"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Input *os.File `argument:"input" required:"true" fileMode:"read" allowSpecial:"false" description:"File to print."`
}

func main() {
	cli.NewApplication[options]().
		WithName("files").
		WithDescription("Read a file configured through struct tags.").
		WithHandler(printFile).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func printFile(_ context.Context, opts options) error {
	// The invocation owns this handle and closes it after execution.
	_, err := io.Copy(os.Stdout, opts.Input)
	return err
}
