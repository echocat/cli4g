package main

import (
	"context"
	"io"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Input *os.File `argument:"input" required:"true" fileMode:"read" allowSpecial:"false" allowStdio:"true" description:"File to print (- or stdin for standard input)."`
}

func main() {
	cli.NewApplication[options]("files", "Read a file configured through struct tags.").
		WithHandler(printFile).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func printFile(ctx context.Context, opts options) error {
	// The invocation owns this handle and closes it after execution.
	_, err := io.Copy(cli.StdoutOfContext(ctx), opts.Input)
	return err
}
