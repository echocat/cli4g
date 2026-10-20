package cli

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/echocat/cli4g/codec"
)

// DefaultCompletionCommand is the ordinary command requirement selected by
// ApplicationRequirement.WithCompletion(true). Its initial value is a ready-to-use
// NewCompletionCommand. Replace this variable before configuring applications,
// without concurrent modification. The shortcut captures its current value;
// Create compiles an independent definition and never mutates the requirement.
// For per-application changes, use a fresh NewCompletionCommand or your own
// CommandRequirement through WithCompletionCommand instead of modifying the
// shared default builder.
var DefaultCompletionCommand CommandRequirementLike = NewCompletionCommand()

// DefaultCompletionRequestCommandName is the special protocol entry selected by
// WithCompletion(true). Replace it before configuring applications, without
// concurrent modification. Empty disables the request entry. Installed adapters
// depend on the captured name, independently of the public command.
var DefaultCompletionRequestCommandName = "__complete"

// CompletionInstallOptions configures the standard completion install command.
// Shell is bash, nu, powershell or zsh. Directory overrides the shell's default
// installation path; for PowerShell and Zsh, it also skips startup-file setup.
// These are ordinary command options, parsed and validated through the usual CLI
// lifecycle together with the selected application and command scopes.
type CompletionInstallOptions struct {
	Shell     string `argument:"shell" enum:"bash|nu|powershell|zsh" required:"true" description:"Shell to configure."`
	Directory string `argument:"directory" type:"completion-directory" description:"Installation directory; omitted uses the shell's default."`
}

// NewCompletionCommand builds a fresh, ordinary command tree named "completion".
// It supports "completion <bash|nu|powershell|zsh>" and
// "completion install <bash|nu|powershell|zsh> [directory]". WithName, WithAliases,
// handlers and all other ordinary CommandRequirement configuration are available.
// Parent requirements, parsing,
// profiles, initialization, execution, cleanup and depth limits apply normally.
// Only the independently configured request entry bypasses that lifecycle.
//
// The command accesses the application's completion API through
// CompletionApplicationFromContext, which custom management commands may also use.
func NewCompletionCommand() *CommandRequirement[NoOptions] {
	command := NewSimpleCommand("completion", "Manage shell completion.").WithHandler(completionGroupHelp)
	for _, shell := range []CompletionShell{CompletionBash, CompletionNushell, CompletionPowerShell, CompletionZsh} {
		command.WithCommand(NewSimpleCommand(string(shell), "Print the "+string(shell)+" adapter.").
			WithHandler(func(ctx context.Context, _ NoOptions) error {
				application, found := CompletionApplicationFromContext(ctx)
				if !found {
					return fmt.Errorf("completion application is unavailable")
				}
				script, err := application.CompletionScript(shell)
				if err != nil {
					return err
				}
				return Print(ctx, script)
			}))
	}
	command.WithCommand(NewCommand[CompletionInstallOptions]("install", "Install a shell adapter.").
		WithCodec(codec.WithCompletionHandler(codec.NewString(), func(ctx context.Context, _ codec.CompletionInput) (codec.CompletionResult, error) {
			return codec.CompletionResult{Handled: true, Directories: true}, ctx.Err()
		}), "completion-directory").
		WithHandler(func(ctx context.Context, options CompletionInstallOptions) error {
			application, found := CompletionApplicationFromContext(ctx)
			if !found {
				return fmt.Errorf("completion application is unavailable")
			}
			path, err := application.InstallCompletion(ctx, CompletionShell(options.Shell), options.Directory)
			if err != nil {
				return err
			}
			return Printf(ctx, "Installed completion: %s\n", path)
		}))
	return command
}

// CompletionApplication exposes a typed application's completion operations to
// ordinary command handlers that do not know the root options type. It is
// implemented by Application, not by callers. Use CompletionApplicationFromContext
// to write custom management commands using the standard renderer/installer.
type CompletionApplication interface {
	Container
	Complete(context.Context, *ParseOpts, CompletionRequest) (codec.CompletionResult, error)
	CompletionScript(CompletionShell) (string, error)
	InstallCompletion(context.Context, CompletionShell, string) (string, error)
	CompletionRequestCommandName() string
}

// CompletionApplicationFromContext returns the current invocation's application
// completion API. It neither constructs options nor creates a new invocation.
// It is available in ordinary lifecycle/command handlers, including a custom
// completion command. Missing invocation data returns nil, false.
func CompletionApplicationFromContext(ctx context.Context) (CompletionApplication, bool) {
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return nil, false
	}
	application, found := invocation.Root().(CompletionApplication)
	return application, found
}

// CompletionCommand returns the ordinary compiled command configured through
// ApplicationRequirement.CompletionCommand, or nil when none was configured.
// It is the same definition returned by Commands/Command and remains readable
// after Close, like all other command metadata.
func (this *Application[O]) CompletionCommand() CommandLike { return this.definition.completionCommand }

// CompletionCommandName returns the configured command's primary name, or empty
// when absent. The name belongs to the command definition, not a separate setting.
func (this *Application[O]) CompletionCommandName() string {
	if command := this.CompletionCommand(); command != nil {
		return command.Name()
	}
	return ""
}

// CompletionRequestCommandName returns the captured special request name. Empty
// means disabled. No implicit default is substituted by Create.
func (this *Application[O]) CompletionRequestCommandName() string {
	return this.definition.completionRequestCommandName
}

func validateCompletionRequestName(name string) error {
	if strings.HasPrefix(name, "-") || strings.ContainsAny(name, "=\x00/\\") || strings.ContainsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) {
		return fmt.Errorf("invalid completion request name %q: expected a single command word", name)
	}
	return nil
}

func completionGroupHelp(ctx context.Context, _ NoOptions) error {
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return fmt.Errorf("completion invocation is unavailable")
	}
	current, found := ContainerFromContext(ctx)
	if !found {
		return fmt.Errorf("completion command is unavailable")
	}
	if invocation.HelpInfo().Target != current {
		return nil // Ordinary ancestor handlers also run before selected children.
	}
	if handler := invocation.HelpHandler(); handler != nil {
		return handler(ctx, invocation)
	}
	return fmt.Errorf("help handler is unavailable")
}
