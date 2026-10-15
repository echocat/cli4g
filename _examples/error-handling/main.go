package main

import (
	"context"
	"errors"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Fail    bool   `flag:"fail" description:"Simulate an operational failure."`
	Message string `argument:"message" default:"Hello" description:"Message to print."`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// All exit codes default to zero: the responder prints but returns errors.
	response := cli.NewRespondErrorHandler().WithOutput(os.Stderr).MustCreate()
	err := cli.NewApplication[options]().
		WithName("error-handling").
		WithDescription("Return errors to the caller and choose application exit codes.").
		WithErrorHandler(response).
		WithHandler(printMessage).
		CreateParseAndExecute(context.Background(), nil, args...)
	if err == nil {
		return 0
	}
	if cli.IsValidationError(err) {
		return 64
	}
	return 70
}

func printMessage(ctx context.Context, opts options) error {
	if opts.Fail {
		return errors.New("simulated execution failure")
	}
	return cli.Println(ctx, opts.Message)
}
