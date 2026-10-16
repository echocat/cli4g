package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type options struct {
	Uppercase bool `flag:"uppercase" short:"u" description:"Convert the injected input to uppercase."`
}

func main() {
	// The host explicitly selects process streams at its outermost boundary.
	ctx := codec.ContextWithFallbackToStdio(context.Background(), true)
	var output bytes.Buffer
	// The embedded application sees only these streams, not the host's stdio.
	embedded := codec.ContextWithStdio(ctx, strings.NewReader("Injected input.\n"), &output, codec.StderrOfContext(ctx))
	cli.NewApplication[options]("stdio", "Run with context-provided in-memory streams and no stdio fallback.").
		WithFallbackToStdio(false).
		WithHandler(copyInput).
		MustCreateParseAndExecute(embedded, nil, os.Args[1:]...)
	if _, err := io.Copy(codec.StdoutOfContext(ctx), &output); err != nil {
		panic(err)
	}
}

func copyInput(ctx context.Context, opts options) error {
	stdin, stdout, _ := codec.StdioOfContext(ctx)
	if stdin == nil || stdout == nil {
		return fmt.Errorf("stdin and stdout must be supplied in the context")
	}
	if opts.Uppercase {
		content, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, strings.ToUpper(string(content)))
		return err
	}
	_, err := io.Copy(stdout, stdin)
	return err
}
