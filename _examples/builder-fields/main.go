package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type options struct {
	Greeting string
	Name     string
}

func main() {
	cli.NewApplication[options]("builder-fields", "Define flags and arguments explicitly", "instead of using tags.").
		WithFlag(cli.NewFlag(codec.NewString(), "greeting", "Greeting to use.").
			WithField("Greeting").
			WithShort('g').
			WithDefault("Hello")).
		WithArgument(cli.NewArgument(codec.NewString(), "name", "Name to greet.").
			WithField("Name").WithRequired(true)).
		WithHandler(sayHello).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func sayHello(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "%s, %s!\n", opts.Greeting, opts.Name)
}
