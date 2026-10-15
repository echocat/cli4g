package benchmarks

import (
	"context"
	"flag"
	"fmt"
	"strconv"
)

// flag has no native int32 helper; use the standard flag.Value extension point.
type int32Value struct{ target *int32 }

func (value int32Value) String() string {
	return strconv.FormatInt(int64(*value.target), 10)
}

func (value int32Value) Set(input string) error {
	parsed, err := strconv.ParseInt(input, 10, 32)
	if err != nil {
		return err
	}
	*value.target = int32(parsed)
	return nil
}

func newFlagRootDirect(result *recorder) (runner, error) {
	flags := flag.NewFlagSet("bench", flag.ContinueOnError)
	var count int32 = 5
	name := flags.String("name", "default", "")
	verbose := flags.Bool("verbose", false, "")
	flags.Var(int32Value{&count}, "count", "")
	used := false
	return runner{
		run: func(_ context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *flag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			if err := flags.Parse(args[1:]); err != nil {
				return err
			}
			return result.accept(values{count, *name, *verbose})
		},
	}, nil
}

func newFlagRootPointerBindings(result *recorder) (runner, error) {
	state := &values{Count: 5, Name: "default"}
	flags := flag.NewFlagSet("bench", flag.ContinueOnError)
	flags.Var(int32Value{&state.Count}, "count", "")
	flags.StringVar(&state.Name, "name", "default", "")
	flags.BoolVar(&state.Verbose, "verbose", false, "")
	used := false
	return runner{
		run: func(_ context.Context, args []string) error {
			var resetErr error
			if used {
				flags.VisitAll(func(flag *flag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			if err := flags.Parse(args[1:]); err != nil {
				return err
			}
			return result.accept(*state)
		},
	}, nil
}

func newFlagSubcommandDirect(result *recorder) (runner, error) {
	flags := flag.NewFlagSet("b", flag.ContinueOnError)
	var count int32 = 5
	name := flags.String("name", "default", "")
	verbose := flags.Bool("verbose", false, "")
	flags.Var(int32Value{&count}, "count", "")
	used := false
	return runner{
		run: func(_ context.Context, args []string) error {
			if len(args) < 3 || args[1] != "a" || args[2] != "b" {
				return fmt.Errorf("expected a b")
			}
			var resetErr error
			if used {
				flags.VisitAll(func(flag *flag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			if err := flags.Parse(args[3:]); err != nil {
				return err
			}
			return result.accept(values{count, *name, *verbose})
		},
	}, nil
}

func newFlagSubcommandPointerBindings(result *recorder) (runner, error) {
	state := &values{Count: 5, Name: "default"}
	flags := flag.NewFlagSet("b", flag.ContinueOnError)
	flags.Var(int32Value{&state.Count}, "count", "")
	flags.StringVar(&state.Name, "name", "default", "")
	flags.BoolVar(&state.Verbose, "verbose", false, "")
	used := false
	return runner{
		run: func(_ context.Context, args []string) error {
			if len(args) < 3 || args[1] != "a" || args[2] != "b" {
				return fmt.Errorf("expected a b")
			}
			var resetErr error
			if used {
				flags.VisitAll(func(flag *flag.Flag) {
					if err := flag.Value.Set(flag.DefValue); err != nil {
						resetErr = err
					}
				})
			}
			used = true
			if resetErr != nil {
				return resetErr
			}
			if err := flags.Parse(args[3:]); err != nil {
				return err
			}
			return result.accept(*state)
		},
	}, nil
}
