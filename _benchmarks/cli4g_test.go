package benchmarks

import (
	"context"

	cli "github.com/echocat/cli4g"
)

type cli4gOptions struct {
	Count   int32  `flag:"count" default:"5"`
	Name    string `flag:"name" default:"default"`
	Verbose bool   `flag:"verbose"`
}

func newCLI4GRootDirect(result *recorder) (runner, error) {
	app, err := cli.NewSimpleApplication().
		WithName("bench").
		WithInt32Flag("count", func(flag *cli.FlagRequirement[int32, int32]) {
			flag.WithDefault(5)
		}).
		WithStringFlag("name", func(flag *cli.FlagRequirement[string, string]) {
			flag.WithDefault("default")
		}).
		WithBoolFlag("verbose").
		WithHandler(func(ctx context.Context, _ cli.NoOptions) error {
			inv, _ := cli.InvocationFromContext(ctx)
			_, count := inv.Flag("count")
			_, name := inv.Flag("name")
			_, verbose := inv.Flag("verbose")
			return result.accept(values{count.(int32), name.(string), verbose == true})
		}).
		Create()
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(ctx context.Context, args []string) error {
			return app.ParseAndExecute(ctx, nil, args[1:]...)
		},
		close: app.Close,
	}, nil
}

func newCLI4GRootStructTags(result *recorder) (runner, error) {
	app, err := cli.NewApplication[cli4gOptions]().
		WithName("bench").
		WithHandler(func(_ context.Context, options cli4gOptions) error {
			return result.accept(values(options))
		}).
		Create()
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(ctx context.Context, args []string) error {
			return app.ParseAndExecute(ctx, nil, args[1:]...)
		},
		close: app.Close,
	}, nil
}

func newCLI4GSubcommandDirect(result *recorder) (runner, error) {
	leaf := cli.NewSimpleCommand().
		WithName("b").
		WithInt32Flag("count", func(flag *cli.FlagRequirement[int32, int32]) {
			flag.WithDefault(5)
		}).
		WithStringFlag("name", func(flag *cli.FlagRequirement[string, string]) {
			flag.WithDefault("default")
		}).
		WithBoolFlag("verbose").
		WithHandler(func(ctx context.Context, _ cli.NoOptions) error {
			inv, _ := cli.InvocationFromContext(ctx)
			_, count := inv.Flag("count")
			_, name := inv.Flag("name")
			_, verbose := inv.Flag("verbose")
			return result.accept(values{count.(int32), name.(string), verbose == true})
		})
	app, err := cli.NewSimpleApplication().
		WithName("bench").
		WithCommand(cli.NewSimpleCommand().
			WithName("a").
			WithCommand(leaf)).
		Create()
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(ctx context.Context, args []string) error {
			return app.ParseAndExecute(ctx, nil, args[1:]...)
		},
		close: app.Close,
	}, nil
}

func newCLI4GSubcommandStructTags(result *recorder) (runner, error) {
	leaf := cli.NewCommand[cli4gOptions]().
		WithName("b").
		WithHandler(func(_ context.Context, options cli4gOptions) error {
			return result.accept(values(options))
		})
	app, err := cli.NewSimpleApplication().
		WithName("bench").
		WithCommand(cli.NewSimpleCommand().
			WithName("a").
			WithCommand(leaf)).
		Create()
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(ctx context.Context, args []string) error {
			return app.ParseAndExecute(ctx, nil, args[1:]...)
		},
		close: app.Close,
	}, nil
}
