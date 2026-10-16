package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/echocat/cli4g"
)

type applicationOptions struct {
	Verbose bool `flag:"verbose" short:"v" description:"Print the selected command path."`
}

type greetOptions struct {
	Name string `argument:"name" required:"true" description:"Name to greet."`
}

type sumOptions struct {
	Numbers []int `argument:"numbers" required:"true" description:"Numbers to add."`
}

func main() {
	cli.NewApplication[*applicationOptions]("commands-introspection", "Inspect the invocation from a root handler.").
		WithHandler(showCommand).
		WithCommand(
			cli.NewCommand[greetOptions]("greet", "Greet someone.").
				WithAliases("hello").
				WithHandler(greet),
			cli.NewSimpleCommand("math", "Arithmetic commands.").
				WithCommand(cli.NewCommand[sumOptions]("sum", "Add one or more integers.").
					WithAliases("add").
					WithHandler(sum),
				),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showCommand(ctx context.Context, opts *applicationOptions) error {
	invocation, _ := cli.InvocationFromContext(ctx)
	path := invocation.NameStack()
	if len(invocation.HelpInfo().Commands) > 0 {
		return invocation.HelpHandler()(ctx, invocation)
	}
	if opts.Verbose {
		_, err := fmt.Fprintln(os.Stderr, "Executing:", strings.Join(path, " "))
		return err
	}
	return nil
}

func greet(ctx context.Context, opts greetOptions) error {
	return cli.Printf(ctx, "Hello, %s!\n", opts.Name)
}

func sum(ctx context.Context, opts sumOptions) error {
	total := 0
	for _, number := range opts.Numbers {
		total += number
	}
	return cli.Printf(ctx, "Sum: %d\n", total)
}
