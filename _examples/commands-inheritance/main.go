package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type commonOptions struct {
	ExitCode int `flag:"exitCode" description:"Will use this code in case of success."`
}

type applicationOptions struct {
	Greeting string `flag:"greeting" default:"Hello" description:"Greeting to use."`
}

type messageOptions struct {
	Punctuation string `flag:"punctuation" default:"!" description:"Punctuation to append."`
}

type greetOptions struct {
	commonOptions
	Application *applicationOptions `parent:"application"`
	Message     *messageOptions     `parent:"matchingType"`
	Name        string              `argument:"name" required:"true" description:"Name to greet."`
}

func main() {
	cli.NewApplication[*applicationOptions]().
		WithName("commands-inheritance").
		WithDescription("Read application and parent command options from a child.").
		WithCommand(cli.NewCommand[*messageOptions]().
			WithName("message").
			WithDescription("Commands using shared message options.").
			WithCommand(cli.NewCommand[greetOptions]().
				WithName("greet").
				WithDescription("Greet someone using parent options.").
				WithHandler(greet),
			),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(ctx context.Context, opts greetOptions) error {
	if err := cli.Printf(ctx, "%s, %s%s\n", opts.Application.Greeting, opts.Name, opts.Message.Punctuation); err != nil {
		return err
	}
	if v := opts.ExitCode; v != 0 {
		os.Exit(v)
	}
	return nil
}
