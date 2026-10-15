package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/echocat/cli4g"
)

type options struct {
	Count uint32 `flag:"count" short:"n" default:"1" description:"Number of greetings (1 to 5)."`
	Name  string `argument:"name" required:"true" description:"Name to greet."`
}

func (opts options) Validate(_ context.Context) error {
	if opts.Count < 1 || opts.Count > 5 {
		return cli.NewValidationErrorf("count must be between 1 and 5")
	}
	if strings.TrimSpace(opts.Name) == "" {
		return cli.NewValidationErrorf("name must not be blank")
	}
	return nil
}

func main() {
	cli.NewApplication[options]().
		WithName("validation").
		WithDescription("Validate decoded options before executing a handler.").
		WithHandler(sayHello).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func sayHello(ctx context.Context, opts options) error {
	return cli.Print(ctx, strings.Repeat(fmt.Sprintf("Hello, %s!\n", opts.Name), int(opts.Count)))
}
