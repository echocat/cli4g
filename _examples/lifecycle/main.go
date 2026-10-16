package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/echocat/cli4g"
)

type options struct {
	Message        string `argument:"message" default:"Hello" description:"Message to store and print."`
	FailInitialize bool   `flag:"fail-initialize" description:"Fail after acquiring the temporary file."`
	Fail           bool   `flag:"fail" description:"Fail in the execution handler."`
	file           *os.File
}

func (opts *options) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(opts.Message) == "" {
		return cli.NewValidationErrorf("message must not be blank")
	}
	return nil
}

func (opts *options) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp("", "cli4g-lifecycle-*")
	if err != nil {
		return err
	}
	opts.file = file // Retain ownership before any later operation can fail.
	if _, err := fmt.Fprintln(os.Stderr, "Initialize"); err != nil {
		return err
	}
	if opts.FailInitialize {
		return errors.New("simulated initialization failure")
	}
	_, err = file.WriteString(opts.Message + "\n")
	return err
}

func (opts *options) Close() error {
	if opts.file == nil {
		return nil
	}
	file := opts.file
	opts.file = nil
	closeErr := file.Close()
	removeErr := os.Remove(file.Name())
	_, printErr := fmt.Fprintln(os.Stderr, "Close")
	return errors.Join(closeErr, removeErr, printErr)
}

func main() {
	cli.NewApplication[*options]("lifecycle", "Initialize invocation-owned resources and clean them up.").
		WithHandler(printMessage).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func printMessage(ctx context.Context, opts *options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := cli.ErrPrintln(ctx, "Execute"); err != nil {
		return err
	}
	if opts.Fail {
		return errors.New("simulated execution failure")
	}
	if _, err := opts.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := io.Copy(cli.StdoutOfContext(ctx), opts.file)
	return err
}
