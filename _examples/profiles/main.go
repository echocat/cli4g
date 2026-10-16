package main

import (
	"context"
	"os"
	"strings"

	"github.com/echocat/cli4g"
)

type greeterOptions struct {
	PrincipalName string `flag:"principal" profile:"text" default:"Carl" description:"Name of the principal to greet."`
	PrincipalRank uint16 `flag:"principal" profile:"number" default:"1" description:"Rank of the principal to greet."`
}

type textOptions struct {
	Foo    string `flag:"foo" profile:"text" default:"hello" description:"Text to print."`
	Suffix string `argument:"suffix" profile:"text" default:"!"`
}

type numberOptions struct {
	Foo    int32 `flag:"foo" profile:"number" description:"Number to multiply."`
	Factor int32 `argument:"factor" profile:"number" default:"1"`
}

func main() {
	// This embedding application chooses how to populate ParseOpts. cli4g does
	// not read this variable or provide a special --profile flag itself.
	profiles := []string{"text"}
	if selected := os.Getenv("CLI4G_PROFILES"); selected != "" {
		profiles = strings.Split(selected, ",")
	}
	cli.NewSimpleApplication("profiles", "Select typed command variants through ParseOpts.Profiles.").
		WithCommand(
			cli.NewCommand[greeterOptions]("greet", "Greet someone.").
				WithHandler(func(ctx context.Context, opts greeterOptions) error {
					if cli.ContextContainsProfile(ctx, "number") {
						return cli.Printf(ctx, "Hello, #%d\n", opts.PrincipalRank)
					}
					return cli.Printf(ctx, "Hello, %s\n", opts.PrincipalName)
				}),
			cli.NewCommand[textOptions]("convert", "Print text.").
				WithProfiles("text").
				WithHandler(func(ctx context.Context, opts textOptions) error {
					return cli.Printf(ctx, "%s%s\n", opts.Foo, opts.Suffix)
				}),
			cli.NewCommand[numberOptions]("convert", "Multiply a number.").
				WithProfiles("number").
				WithHandler(func(ctx context.Context, opts numberOptions) error {
					return cli.Printf(ctx, "%d\n", opts.Foo*opts.Factor)
				}),
		).
		MustCreateParseAndExecute(context.Background(), &cli.ParseOpts{
			Profiles: profiles,
		}, os.Args[1:]...)
}
