package cli

// DefaultApplicationHelpTemplate renders complete application help.
const DefaultApplicationHelpTemplate = `{{.FullName}}{{if .Description}} - {{.Description}}{{end}}

{{template "usage" .}}{{if .Commands}}
Commands:
{{range .Commands}}{{template "command-entry" .}}{{end}}{{end}}{{if .Arguments}}
Arguments:
{{range .Arguments}}{{template "argument" .}}{{end}}{{end}}{{if .Flags}}
Flags:
{{range .Flags}}{{template "flag" .}}{{end}}{{end}}{{if .InheritedFlags}}
Inherited flags:
{{range .InheritedFlags}}{{template "flag" .}}{{end}}{{end}}`

// DefaultCommandHelpTemplate renders complete selected-command help.
const DefaultCommandHelpTemplate = DefaultApplicationHelpTemplate

// DefaultUsageHelpTemplate renders invocation syntax.
const DefaultUsageHelpTemplate = `Usage:
  {{.Usage}}
`

// DefaultCommandEntryHelpTemplate renders one available command.
const DefaultCommandEntryHelpTemplate = `  {{.Label}}{{if .Description}}	{{.Description}}{{end}}
`

// DefaultFlagHelpTemplate renders one own or inherited flag.
const DefaultFlagHelpTemplate = `  {{.Label}}{{if .Description}}	{{.Description}}{{end}}{{if .Required}} (required){{end}}{{if .EnvVars}} (env: {{join .EnvVars ", "}}){{end}}{{if .HasDefault}} (default: {{.DefaultText}}){{end}}{{if and .Choices (not .HasChoiceDescriptions)}} (choices: {{.ChoicesText}}){{end}}
{{if .HasChoiceDescriptions}}{{range .Choices}}    {{.Text}}{{if .Description}}	{{.Description}}{{end}}
{{end}}{{end}}`

// DefaultArgumentHelpTemplate renders one positional argument.
const DefaultArgumentHelpTemplate = `  {{.Label}}{{if .Description}}	{{.Description}}{{end}}{{if .Required}} (required){{end}}{{if .HasDefault}} (default: {{.DefaultText}}){{end}}{{if and .Choices (not .HasChoiceDescriptions)}} (choices: {{.ChoicesText}}){{end}}
{{if .HasChoiceDescriptions}}{{range .Choices}}    {{.Text}}{{if .Description}}	{{.Description}}{{end}}
{{end}}{{end}}`
