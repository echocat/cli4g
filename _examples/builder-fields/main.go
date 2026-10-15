package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Greeting string
	Name     string
}

func main() {
	cli.NewApplication[options]().
		WithName("builder-fields").
		WithDescription("Define flags and arguments explicitly instead of using tags.").
		WithStringFlag("greeting", func(f *cli.FlagRequirement[string, string]) {
			f.WithField("Greeting").
				WithShort('g').
				WithDefault("Hello").
				WithDescription("Greeting to use.")
		}).
		WithStringArgument("name", func(a *cli.ArgumentRequirement[string, string]) {
			a.WithField("Name").
				WithRequired(true).
				WithDescription("Name to greet.")
		}).
		WithHandler(sayHello).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func sayHello(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "%s, %s!\n", opts.Greeting, opts.Name)
}
