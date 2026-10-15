package benchmarks

import (
	"context"

	urfave "github.com/urfave/cli/v3"
)

func newUrfaveRootDirect(result *recorder) (runner, error) {
	root := &urfave.Command{
		Name: "bench",
		Flags: []urfave.Flag{
			&urfave.Int32Flag{Name: "count", Value: 5},
			&urfave.StringFlag{Name: "name", Value: "default"},
			&urfave.BoolFlag{Name: "verbose"},
		},
		Action: func(_ context.Context, command *urfave.Command) error {
			return result.accept(values{
				command.Int32("count"),
				command.String("name"),
				command.Bool("verbose"),
			})
		},
	}
	return runner{run: root.Run}, nil
}

func newUrfaveRootPointerBindings(result *recorder) (runner, error) {
	state := new(values)
	root := &urfave.Command{
		Name: "bench",
		Flags: []urfave.Flag{
			&urfave.Int32Flag{Name: "count", Value: 5, Destination: &state.Count},
			&urfave.StringFlag{Name: "name", Value: "default", Destination: &state.Name},
			&urfave.BoolFlag{Name: "verbose", Destination: &state.Verbose},
		},
		Action: func(_ context.Context, _ *urfave.Command) error {
			return result.accept(*state)
		},
	}
	return runner{run: root.Run}, nil
}

func newUrfaveSubcommandDirect(result *recorder) (runner, error) {
	leaf := &urfave.Command{
		Name: "b",
		Flags: []urfave.Flag{
			&urfave.Int32Flag{Name: "count", Value: 5},
			&urfave.StringFlag{Name: "name", Value: "default"},
			&urfave.BoolFlag{Name: "verbose"},
		},
		Action: func(_ context.Context, command *urfave.Command) error {
			return result.accept(values{
				command.Int32("count"),
				command.String("name"),
				command.Bool("verbose"),
			})
		},
	}
	root := &urfave.Command{
		Name: "bench",
		Commands: []*urfave.Command{
			{Name: "a", Commands: []*urfave.Command{leaf}},
		},
	}
	return runner{run: root.Run}, nil
}

func newUrfaveSubcommandPointerBindings(result *recorder) (runner, error) {
	state := new(values)
	leaf := &urfave.Command{
		Name: "b",
		Flags: []urfave.Flag{
			&urfave.Int32Flag{Name: "count", Value: 5, Destination: &state.Count},
			&urfave.StringFlag{Name: "name", Value: "default", Destination: &state.Name},
			&urfave.BoolFlag{Name: "verbose", Destination: &state.Verbose},
		},
		Action: func(_ context.Context, _ *urfave.Command) error {
			return result.accept(*state)
		},
	}
	root := &urfave.Command{
		Name: "bench",
		Commands: []*urfave.Command{
			{Name: "a", Commands: []*urfave.Command{leaf}},
		},
	}
	return runner{run: root.Run}, nil
}
