package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/echocat/cli4g"
)

type OutputOptions struct {
	Uppercase bool `flag:"uppercase" short:"u" description:"Print in uppercase."`
}

type greetOptions struct {
	OutputOptions
	Name string `argument:"name" required:"true" description:"Name to greet."`
}

type echoOptions struct {
	OutputOptions
	Message string `argument:"message" required:"true" description:"Message to print."`
}

func main() {
	cli.NewSimpleApplication().
		WithName("embedded-options").
		WithDescription("Reuse a tagged options group through struct embedding.").
		WithCommand(
			cli.NewCommand[greetOptions]().WithName("greet").WithDescription("Greet someone.").WithHandler(greet),
			cli.NewCommand[echoOptions]().WithName("echo").WithDescription("Print a message.").WithHandler(echo),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(_ context.Context, opts greetOptions) error {
	return printMessage(opts.OutputOptions, "Hello, "+opts.Name+"!")
}

func echo(_ context.Context, opts echoOptions) error {
	return printMessage(opts.OutputOptions, opts.Message)
}

func printMessage(opts OutputOptions, message string) error {
	if opts.Uppercase {
		message = strings.ToUpper(message)
	}
	_, err := fmt.Fprintln(os.Stdout, message)
	return err
}
