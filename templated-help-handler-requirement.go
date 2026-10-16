package cli

import (
	"io"
	"text/template"
)

// NewTemplatedHelpHandler starts an empty help-template builder.
// Set templates explicitly, using the Default*HelpTemplate constants as needed.
func NewTemplatedHelpHandler() *TemplatedHelpHandlerRequirement {
	return &TemplatedHelpHandlerRequirement{}
}

// TemplatedHelpHandlerRequirement configures text/template-based help.
// Empty templates stay empty; defaults are not inserted automatically.
// Main templates receive [HelpTemplateData]; entry templates receive the matching
// Help*TemplateData type. Built-in functions include join and quote.
// Create reports invalid syntax; execution reports render/write/default errors.
// Formatting is enabled by default: tabbed runs form tables with independent
// column widths per indentation level. Long lines wrap with hanging indentation.
// Widths count Unicode code points, not terminal cells or ANSI escape sequences.
// Unbreakable words and labels are preserved even when wider than the output.
type TemplatedHelpHandlerRequirement struct {
	// Output remains caller-owned. Custom writers and handlers need their own synchronization.
	Output   io.Writer // nil selects context stderr, or io.Discard when unavailable
	ExitCode int       // non-zero calls os.Exit after successful rendering/writing

	// Width overrides the output width in characters. Zero detects terminal width
	// from the selected output's Fd method, falling back to 80. Negative is invalid.
	// Detection occurs for each rendering, including when context streams change.
	Width int

	ApplicationTemplate  string           // template "application": root help
	CommandTemplate      string           // template "command": selected command help
	UsageTemplate        string           // template "usage": invocation syntax
	CommandEntryTemplate string           // template "command-entry": available command
	FlagTemplate         string           // template "flag": flag entry
	ArgumentTemplate     string           // template "argument": positional entry
	Funcs                template.FuncMap // custom functions must be concurrent-safe

	formattingDisabled bool
}

// WithOutput selects a caller-owned writer. Nil uses context stderr at each
// invocation, or io.Discard when no output is supplied by the context.
func (this *TemplatedHelpHandlerRequirement) WithOutput(v io.Writer) *TemplatedHelpHandlerRequirement {
	this.SetOutput(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetOutput(v io.Writer) { this.Output = v }

func (this *TemplatedHelpHandlerRequirement) WithExitCode(v int) *TemplatedHelpHandlerRequirement {
	this.SetExitCode(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetExitCode(v int) { this.ExitCode = v }

// WithWidth sets a fixed help width. Zero detects terminal width, falling back to
// 80 for non-terminals, writers without Fd, and unsuccessful size queries.
func (this *TemplatedHelpHandlerRequirement) WithWidth(v int) *TemplatedHelpHandlerRequirement {
	this.SetWidth(v)
	return this
}

func (this *TemplatedHelpHandlerRequirement) SetWidth(v int) { this.Width = v }

// WithFormatting enables automatic column alignment and word wrapping. Enabled
// by default. Disable it for byte-exact or deliberately preformatted templates;
// disabling also skips terminal-size queries. Create snapshots this setting.
func (this *TemplatedHelpHandlerRequirement) WithFormatting(v bool) *TemplatedHelpHandlerRequirement {
	this.SetFormatting(v)
	return this
}

func (this *TemplatedHelpHandlerRequirement) SetFormatting(v bool) { this.formattingDisabled = !v }

func (this *TemplatedHelpHandlerRequirement) WithApplicationTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetApplicationTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetApplicationTemplate(v string) {
	this.ApplicationTemplate = v
}

func (this *TemplatedHelpHandlerRequirement) WithCommandTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetCommandTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetCommandTemplate(v string) { this.CommandTemplate = v }

func (this *TemplatedHelpHandlerRequirement) WithUsageTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetUsageTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetUsageTemplate(v string) { this.UsageTemplate = v }

func (this *TemplatedHelpHandlerRequirement) WithCommandEntryTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetCommandEntryTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetCommandEntryTemplate(v string) {
	this.CommandEntryTemplate = v
}

func (this *TemplatedHelpHandlerRequirement) WithFlagTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetFlagTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetFlagTemplate(v string) { this.FlagTemplate = v }

func (this *TemplatedHelpHandlerRequirement) WithArgumentTemplate(v string) *TemplatedHelpHandlerRequirement {
	this.SetArgumentTemplate(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetArgumentTemplate(v string) {
	this.ArgumentTemplate = v
}

func (this *TemplatedHelpHandlerRequirement) WithFuncs(v template.FuncMap) *TemplatedHelpHandlerRequirement {
	this.SetFuncs(v)
	return this
}
func (this *TemplatedHelpHandlerRequirement) SetFuncs(v template.FuncMap) { this.Funcs = v }
