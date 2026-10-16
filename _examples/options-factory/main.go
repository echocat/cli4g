package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
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
	cli.NewApplication[*options]("options-factory", "Create options from configuration and inject a service.").
		WithCreateOptionsHandler(func(ctx context.Context) (*options, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			// Bind each invocation's service to its own context output.
			service := &greetingService{output: codec.StdoutOfContext(ctx)}
			return &options{Greeting: config.Greeting, Service: service}, nil
		}).
		WithHandler(greet).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(_ context.Context, opts *options) error {
	return opts.Service.Greet(opts.Greeting, opts.Name)
}
