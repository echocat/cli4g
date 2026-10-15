package benchmarks

import (
	"context"

	"github.com/alecthomas/kong"
)

type kongLeaf struct {
	Count   int32  `default:"5"`
	Name    string `default:"default"`
	Verbose bool
}

func (options *kongLeaf) Run(result *recorder) error {
	return result.accept(values{options.Count, options.Name, options.Verbose})
}

type kongCommands struct {
	A struct {
		B kongLeaf `cmd:""`
	} `cmd:""`
}

func newKongRootStructTags(result *recorder) (runner, error) {
	definition, err := kong.New(
		new(kongLeaf),
		kong.Name("bench"),
		kong.Bind(result),
	)
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(_ context.Context, args []string) error {
			parsed, err := definition.Parse(args[1:])
			if err != nil {
				return err
			}
			return parsed.Run()
		},
	}, nil
}

func newKongSubcommandStructTags(result *recorder) (runner, error) {
	definition, err := kong.New(
		new(kongCommands),
		kong.Name("bench"),
		kong.Bind(result),
	)
	if err != nil {
		return runner{}, err
	}
	return runner{
		run: func(_ context.Context, args []string) error {
			parsed, err := definition.Parse(args[1:])
			if err != nil {
				return err
			}
			return parsed.Run()
		},
	}, nil
}
