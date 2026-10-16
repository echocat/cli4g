package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

const applicationHelp = `Help for {{.FullName}}
{{.Description}}

{{template "usage" .}}{{if .Arguments}}
Inputs:
{{range .Arguments}}{{template "argument" .}}{{end}}{{end}}{{if .Flags}}
Options:
{{range .Flags}}{{template "flag" .}}{{end}}{{end}}`

const flagHelp = `  {{.Label}}:	{{.Description}}{{if .HasDefault}} [default: {{.DefaultText}}]{{end}}
`

const argumentHelp = `  {{.Label}}:	{{.Description}}{{if .Required}} [required]{{end}}
`

type options struct {
	Greeting string `flag:"greeting" short:"g" default:"Hello" description:"Greeting to use."`
	Name     string `argument:"name" required:"true" description:"Name to greet."`
}

func main() {
	cli.NewApplication[options]("custom-help", "A greeting with a custom help layout.").
		WithHelpHandler(cli.NewTemplatedHelpHandler().
			WithOutput(os.Stdout).
			WithApplicationTemplate(applicationHelp).
			WithUsageTemplate("Run: {{.Usage}}\n").
			WithFlagTemplate(flagHelp).
			WithArgumentTemplate(argumentHelp).
			MustCreate(),
		).
		WithHandler(greet).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "%s, %s!\n", opts.Greeting, opts.Name)
}
