package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/echocat/cli4g"
)

type options struct {
	Value    string `argument:"value" default:"42" description:"An integer checked by Validate."`
	FailRead bool   `flag:"fail-read" description:"Simulate a technical read failure during Validate."`
}

func (opts options) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.FailRead {
		return fmt.Errorf("reading configuration: %w", io.ErrUnexpectedEOF)
	}
	if _, err := strconv.Atoi(opts.Value); err != nil {
		return cli.NewValidationError("value must be an integer", err)
	}
	return nil
}

func main() {
	response := cli.DefaultErrorHandler
	cli.NewApplication[options]("validation-errors", "Distinguish invalid input from technical failures in Validate.").
		WithErrorHandler(func(ctx context.Context, invocation cli.InvocationLike, err error) error {
			var number *strconv.NumError
			var detail string
			switch {
			case cli.IsValidationError(err) && errors.As(err, &number):
				detail = "Expected: an integer"
			case errors.Is(err, io.ErrUnexpectedEOF):
				detail = "Configuration could not be read completely"
			}
			if detail != "" {
				_, printErr := fmt.Fprintln(os.Stderr, detail)
				err = errors.Join(err, printErr)
			}
			return response(ctx, invocation, err)
		}).
		WithHandler(showValue).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showValue(ctx context.Context, opts options) error {
	return cli.Println(ctx, "Accepted:", opts.Value)
}
