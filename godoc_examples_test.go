package cli_test

import (
	"context"
	"fmt"

	cli "github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

func ExampleNewApplication() {
	type options struct {
		Name     string `argument:"name" required:"true"`
		Greeting string `flag:"greeting" short:"g" default:"Hello"`
	}
	cli.NewApplication[options]("greet", "Greet someone.").
		WithHandler(func(ctx context.Context, opts options) error {
			_, err := fmt.Fprintf(codec.StdoutOfContext(ctx), "%s, %s!\n", opts.Greeting, opts.Name)
			return err
		}).MustCreateParseAndExecute(context.Background(), nil, "Ada")
	// Output: Hello, Ada!
}

func ExampleApplication_Parse() {
	app := cli.NewSimpleApplication("labels").
		WithErrorHandlerEnabled(false).
		WithStringSliceFlag("label").
		MustCreate()
	defer func() {
		if err := app.Close(); err != nil {
			panic(err)
		}
	}()
	invocation, err := app.Parse(context.Background(), nil, "--label=blue", "--label=green")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := invocation.Close(); err != nil {
			panic(err)
		}
	}()
	_, labels := invocation.Flag("label")
	fmt.Println(labels)
	// Output: [blue green]
}

func ExampleFlagRequirement_WithField() {
	type options struct{ Port uint32 }
	app := cli.NewApplication[options]("server").
		WithUint32Flag("port", func(flag *cli.FlagRequirement[uint32, uint32]) {
			flag.
				WithField("Port").
				WithDefault(8080)
		}).
		MustCreate()
	defer func() {
		if err := app.Close(); err != nil {
			panic(err)
		}
	}()
	invocation, err := app.Parse(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := invocation.Close(); err != nil {
			panic(err)
		}
	}()
	_, opts := invocation.Application()
	fmt.Println(opts.Port)
	// Output: 8080
}

func ExampleNewValidationErrorf() {
	err := cli.NewValidationErrorf("port must be positive")
	fmt.Println(cli.IsValidationError(err))
	fmt.Println(err)
	// Output:
	// true
	// port must be positive
}

func ExampleNewCommand() {
	type options struct {
		Name string `argument:"name" default:"Ada"`
	}
	command := cli.NewCommand[options]("hello", "Greet", "someone.").
		WithHandler(func(ctx context.Context, opts options) error {
			_, err := fmt.Fprintln(codec.StdoutOfContext(ctx), "Hello, "+opts.Name+"!")
			return err
		})
	cli.NewSimpleApplication("app").
		WithCommand(command).
		MustCreateParseAndExecute(context.Background(), nil, "hello", "Grace")
	// Output: Hello, Grace!
}

func ExampleNewFlag() {
	flag := cli.NewFlag(codec.NewString(), "greeting", "Greeting to use.").
		WithAliases("message").
		WithDefault("Hello")
	app := cli.NewSimpleApplication("greet").
		WithFlag(flag).
		MustCreate()
	defer func() {
		if err := app.Close(); err != nil {
			panic(err)
		}
	}()
	invocation := app.MustParse(context.Background(), nil, "--message=Hey")
	defer func() {
		if err := invocation.Close(); err != nil {
			panic(err)
		}
	}()
	_, value := invocation.Flag("greeting")
	fmt.Println(value)
	// Output: Hey
}

func ExampleNewArgument() {
	argument := cli.NewArgument(codec.NewString(), "name", "Name", "to greet.").
		WithRequired(true)
	app := cli.NewSimpleApplication("greet").
		WithArgument(argument).
		MustCreate()
	defer func() {
		if err := app.Close(); err != nil {
			panic(err)
		}
	}()
	invocation := app.MustParse(context.Background(), nil, "Ada")
	defer func() {
		if err := invocation.Close(); err != nil {
			panic(err)
		}
	}()
	_, value := invocation.Argument(0)
	fmt.Println(value)
	// Output: Ada
}
