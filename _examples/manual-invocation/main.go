package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Greeting string `flag:"greeting" short:"g" default:"Hello" description:"Greeting to use."`
	Name     string `argument:"name" default:"World" description:"Name to greet."`
	Calls    int
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		if cli.IsValidationError(err) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (err error) {
	app, err := cli.NewApplication[*options]().
		WithName("manual-invocation").
		WithDescription("Create once, then parse, execute, and close independent invocations.").
		WithErrorHandlerEnabled(false).
		WithHandler(greet).
		Create()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, app.Close()) }()

	for _, input := range [][]string{args, {"--greeting", "Hey", "Carl"}} {
		help, err := executeInvocation(ctx, app, input)
		if err != nil {
			return err
		}
		if help {
			break
		}
	}
	return nil
}

func executeInvocation(ctx context.Context, app *cli.Application[*options], args []string) (help bool, err error) {
	invocation, err := app.Parse(ctx, nil, args...)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, invocation.Close()) }()
	return invocation.HelpRequested(), invocation.Execute(ctx)
}

func greet(_ context.Context, opts *options) error {
	opts.Calls++
	_, err := fmt.Printf("%s, %s! (calls=%d)\n", opts.Greeting, opts.Name, opts.Calls)
	return err
}
