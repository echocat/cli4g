package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/echocat/cli4g"
)

type configuration struct {
	Greeting string
}

type greetingService struct {
	output io.Writer
}

func (service *greetingService) Greet(greeting, name string) error {
	_, err := fmt.Fprintf(service.output, "%s, %s!\n", greeting, name)
	return err
}

type options struct {
	Greeting string `flag:"greeting" short:"g" description:"Override the configured greeting."`
	Name     string `argument:"name" default:"World" description:"Name to greet."`
	Service  *greetingService
}

func main() {
	config := configuration{Greeting: "Welcome"}
	service := &greetingService{output: os.Stdout}
	cli.NewApplication[*options]().
		WithName("options-factory").
		WithDescription("Create options from configuration and inject a service.").
		WithCreateOptionsHandler(func(ctx context.Context) (*options, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			// Allocate fresh options for each invocation; share the provided service.
			return &options{Greeting: config.Greeting, Service: service}, nil
		}).
		WithHandler(greet).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(_ context.Context, opts *options) error {
	return opts.Service.Greet(opts.Greeting, opts.Name)
}
