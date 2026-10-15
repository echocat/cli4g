package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Greeting string `flag:"greeting" short:"g" default:"Hello" description:"Greeting to use."`
	Name     string `argument:"name" required:"true" description:"Name of the person to greet."`
}

func main() {
	cli.NewApplication[options]().
		WithName("simple").
		WithDescription("Greet someone using cli4g.").
		WithHandler(sayHello).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func sayHello(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "%s, %s!\n", opts.Greeting, opts.Name)
}
