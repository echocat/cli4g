package main

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Document *document `argument:"input" required:"true" description:"Document to read."`
	Fail     bool      `flag:"fail" description:"Fail after the codec opens the document."`
}

func main() {
	cli.NewApplication[options]().
		WithName("custom-resource-codec").
		WithDescription("Acquire a document through an invocation-owned codec scope.").
		WithCodec(documentCodec{}).
		WithHandler(printDocument).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func printDocument(ctx context.Context, opts options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Fail {
		return errors.New("simulated execution failure")
	}
	_, err := io.Copy(cli.StdoutOfContext(ctx), opts.Document.Reader)
	return err
}
