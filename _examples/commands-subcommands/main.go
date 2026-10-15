package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
)

type numbersOptions struct {
	Numbers []int `argument:"numbers" required:"true" description:"Numbers to use."`
}

func main() {
	cli.NewSimpleApplication().
		WithName("commands-subcommands").
		WithDescription("Commands organized under a command group.").
		WithCommand(cli.NewSimpleCommand().
			WithName("math").
			WithDescription("Arithmetic commands.").
			WithCommand(
				cli.NewCommand[numbersOptions]().
					WithName("sum").
					WithDescription("Add integers.").
					WithHandler(sum),
				cli.NewCommand[numbersOptions]().
					WithName("product").
					WithDescription("Multiply integers.").
					WithHandler(product),
			),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func sum(_ context.Context, opts numbersOptions) error {
	total := 0
	for _, number := range opts.Numbers {
		total += number
	}
	_, err := fmt.Printf("Sum: %d\n", total)
	return err
}

func product(ctx context.Context, opts numbersOptions) error {
	total := 1
	for _, number := range opts.Numbers {
		total *= number
	}
	return cli.Printf(ctx, "Product: %d\n", total)
}
