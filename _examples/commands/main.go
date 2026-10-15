package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type greetOptions struct {
	Name string `argument:"name" required:"true" description:"Name to greet."`
}

type sumOptions struct {
	Numbers []int `argument:"numbers" required:"true" description:"Numbers to add."`
}

func main() {
	cli.NewSimpleApplication().
		WithName("commands").
		WithDescription("Two independent commands.").
		WithCommand(
			cli.NewCommand[greetOptions]().
				WithName("greet").
				WithDescription("Greet someone.").
				WithHandler(greet),
			cli.NewCommand[sumOptions]().
				WithName("sum").
				WithDescription("Add integers.").
				WithHandler(sum),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(_ context.Context, opts greetOptions) error {
	_, err := fmt.Printf("Hello, %s!\n", opts.Name)
	return err
}

func sum(_ context.Context, opts sumOptions) error {
	total := 0
	for _, number := range opts.Numbers {
		total += number
	}
	_, err := fmt.Printf("Sum: %d\n", total)
	return err
}
