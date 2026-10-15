package main

import (
	"context"
	"fmt"
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

func sayHello(_ context.Context, opts options) error {
	_, err := fmt.Printf("%s, %s!\n", opts.Greeting, opts.Name)
	return err
}
