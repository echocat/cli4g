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
	cli.NewSimpleApplication("commands-subcommands", "Commands organized under a command group.").
		WithCommand(cli.NewSimpleCommand("math", "Arithmetic commands.").
			WithCommand(
				cli.NewCommand[numbersOptions]("sum", "Add integers.").
					WithHandler(sum),
				cli.NewCommand[numbersOptions]("product", "Multiply integers.").
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
