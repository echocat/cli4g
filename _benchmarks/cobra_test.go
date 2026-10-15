package benchmarks

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newCobraRootDirect(result *recorder) (runner, error) {
	root := &cobra.Command{Use: "bench"}
	flags := root.Flags()
	flags.Int32("count", 5, "")
	flags.String("name", "default", "")
	flags.Bool("verbose", false, "")
	root.RunE = func(command *cobra.Command, _ []string) error {
		count, err := command.Flags().GetInt32("count")
		if err != nil {
			return err
		}
		name, err := command.Flags().GetString("name")
		if err != nil {
			return err
		}
		verbose, err := command.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		return result.accept(values{count, name, verbose})
	}
	used := false
	return runner{
		run: func(ctx context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *pflag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
					flag.Changed = false
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			root.SetArgs(args[1:])
			return root.ExecuteContext(ctx)
		},
	}, nil
}

func newCobraRootPointerBindings(result *recorder) (runner, error) {
	state := new(values)
	root := &cobra.Command{Use: "bench"}
	flags := root.Flags()
	flags.Int32Var(&state.Count, "count", 5, "")
	flags.StringVar(&state.Name, "name", "default", "")
	flags.BoolVar(&state.Verbose, "verbose", false, "")
	root.RunE = func(_ *cobra.Command, _ []string) error {
		return result.accept(*state)
	}
	used := false
	return runner{
		run: func(ctx context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *pflag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
					flag.Changed = false
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			root.SetArgs(args[1:])
			return root.ExecuteContext(ctx)
		},
	}, nil
}

func newCobraSubcommandDirect(result *recorder) (runner, error) {
	leaf := &cobra.Command{Use: "b"}
	flags := leaf.Flags()
	flags.Int32("count", 5, "")
	flags.String("name", "default", "")
	flags.Bool("verbose", false, "")
	leaf.RunE = func(command *cobra.Command, _ []string) error {
		count, err := command.Flags().GetInt32("count")
		if err != nil {
			return err
		}
		name, err := command.Flags().GetString("name")
		if err != nil {
			return err
		}
		verbose, err := command.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		return result.accept(values{count, name, verbose})
	}
	group := &cobra.Command{Use: "a"}
	group.AddCommand(leaf)
	root := &cobra.Command{Use: "bench"}
	root.AddCommand(group)
	used := false
	return runner{
		run: func(ctx context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *pflag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
					flag.Changed = false
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			root.SetArgs(args[1:])
			return root.ExecuteContext(ctx)
		},
	}, nil
}

func newCobraSubcommandPointerBindings(result *recorder) (runner, error) {
	state := new(values)
	leaf := &cobra.Command{Use: "b"}
	flags := leaf.Flags()
	flags.Int32Var(&state.Count, "count", 5, "")
	flags.StringVar(&state.Name, "name", "default", "")
	flags.BoolVar(&state.Verbose, "verbose", false, "")
	leaf.RunE = func(_ *cobra.Command, _ []string) error {
		return result.accept(*state)
	}
	group := &cobra.Command{Use: "a"}
	group.AddCommand(leaf)
	root := &cobra.Command{Use: "bench"}
	root.AddCommand(group)
	used := false
	return runner{
		run: func(ctx context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *pflag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
					flag.Changed = false
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			root.SetArgs(args[1:])
			return root.ExecuteContext(ctx)
		},
	}, nil
}
