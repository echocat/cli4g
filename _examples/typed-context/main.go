package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type applicationOptions struct {
	Greeting string `flag:"greeting" default:"Hello" description:"Greeting to use."`
	Verbose  bool   `flag:"verbose" short:"v" default:"false" description:"A flag read directly from context."`
}

type messageOptions struct {
	Punctuation string `flag:"punctuation" default:"!" description:"Punctuation to append."`
}

type greetOptions struct {
	Name string `argument:"name" required:"true" description:"Name to greet."`
}

func main() {
	cli.NewApplication[*applicationOptions]().
		WithName("typed-context").
		WithDescription("Read typed invocation values from a context.").
		WithCommand(cli.NewCommand[*messageOptions]().
			WithName("message").
			WithDescription("Message commands.").
			WithCommand(cli.NewCommand[greetOptions]().
				WithName("greet").
				WithDescription("Greet someone using context helpers.").
				WithHandler(greet))).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(ctx context.Context, _ greetOptions) error {
	return printGreeting(ctx)
}

// A helper receiving only context can read the invocation's existing values.
func printGreeting(ctx context.Context) error {
	application, found := cli.ApplicationFromContext[*applicationOptions](ctx)
	if !found {
		return fmt.Errorf("application options are unavailable")
	}
	parent, found := cli.CommandFromContext[*messageOptions](ctx, "message")
	if !found {
		return fmt.Errorf("message options are unavailable")
	}
	current, found := cli.CommandFromContext[greetOptions](ctx)
	if !found {
		return fmt.Errorf("greet options are unavailable")
	}
	verbose, found := cli.FlagFromContext[bool](ctx, "verbose")
	if !found {
		return fmt.Errorf("verbose flag is unavailable")
	}
	_, err := fmt.Printf("%s, %s%s verbose=%t\n", application.Greeting, current.Name, parent.Punctuation, verbose)
	return err
}
