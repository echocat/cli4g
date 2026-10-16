package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func testDefaultTemplatedHelpRequirement() *TemplatedHelpHandlerRequirement {
	return &TemplatedHelpHandlerRequirement{
		ApplicationTemplate:  DefaultApplicationHelpTemplate,
		CommandTemplate:      DefaultCommandHelpTemplate,
		UsageTemplate:        DefaultUsageHelpTemplate,
		CommandEntryTemplate: DefaultCommandEntryHelpTemplate,
		FlagTemplate:         DefaultFlagHelpTemplate,
		ArgumentTemplate:     DefaultArgumentHelpTemplate,
	}
}

func TestTemplatedHelpRendersRootAndCommandMetadataAndCodecDefaults(t *testing.T) {
	type rootOptions struct {
		Name   string `flag:"name|n" short:"N" env:"HELP_NAME|NAME" default:"" description:"Server name." placeholder:"NAME"`
		Secret int    `flag:"secret" visible:"false" default:"invalid"`
	}
	type options struct {
		Port  int      `flag:"port" required:"true" description:"Listen port."`
		Items []string `argument:"items" description:"Input items."`
	}
	var output bytes.Buffer
	handler := testDefaultTemplatedHelpRequirement().
		WithOutput(&output).
		MustCreate()
	app := NewApplication[rootOptions]("app", "Application description.").
		WithHelpHandler(handler).
		WithCommand(NewCommand[options]("run", "Run description.").
			WithAliases("r").
			WithArgument(NewArgument(codec.NewStrings(), "items", "Input items.").
				WithField("Items").
				WithDefaultStr("one two", "three"),
			),
		).
		MustCreate()
	for _, path := range [][]string{nil, {"run"}} {
		output.Reset()
		invocation, err := app.Parse(context.Background(), nil, append(path, "--help")...)
		require.NoError(t, err)
		require.NoError(t, invocation.Execute(nil))
		require.NoError(t, invocation.Close())
		text := output.String()
		assert.NotContains(t, text, "secret")
		assert.Contains(t, text, "-N, --name, --n NAME")
		assert.Contains(t, text, "env: HELP_NAME, NAME")
		assert.Contains(t, text, `default: ""`)
		assert.Contains(t, text, "-h, --help")
		if len(path) == 0 {
			assert.Contains(t, text, "app - Application description.")
			assert.Contains(t, text, "app [flags] [command]")
			assert.Contains(t, text, "run, r")
		} else {
			assert.Contains(t, text, "app run - Run description.")
			assert.Contains(t, text, "app run [flags] [ITEMS...]")
			assert.Contains(t, text, "--port PORT")
			assert.Contains(t, text, "(required)")
			assert.Contains(t, text, "Inherited flags:")
			assert.Contains(t, text, `default: ["one two", three]`)
		}
	}
}

func TestTemplatedHelpOmitsHiddenCommands(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, visibleSibling := range []bool{false, true} {
			t.Run(fmt.Sprintf("nested=%t/visibleSibling=%t", nested, visibleSibling), func(t *testing.T) {
				var output bytes.Buffer
				var calls int
				hidden := NewSimpleCommand("secret", "Hidden command description.").
					WithAliases("secret-alias").
					WithVisible(false).
					WithHandler(func(context.Context, NoOptions) error {
						calls++
						return nil
					})
				commands := []CommandRequirementLike{hidden}
				if visibleSibling {
					commands = append(commands, NewSimpleCommand("public", "Public command."))
				}
				var path []string
				if nested {
					commands = []CommandRequirementLike{NewSimpleCommand("parent").
						WithCommand(commands...),
					}
					path = []string{"parent"}
				}
				app := NewSimpleApplication("app").
					WithHelpHandler(testDefaultTemplatedHelpRequirement().
						WithOutput(&output).
						MustCreate(),
					).
					WithCommand(commands...).
					MustCreate()
				defer func() { assert.NoError(t, app.Close()) }()
				hidden.SetVisible(true) // Compiled commands retain their own visibility snapshot.
				invocation := app.MustParse(context.Background(), nil, append(slices.Clone(path), "--help")...)
				require.NoError(t, invocation.Execute(nil))
				require.NoError(t, invocation.Close())
				assert.NotContains(t, output.String(), "secret")
				assert.NotContains(t, output.String(), "Hidden command description.")
				if visibleSibling {
					assert.Contains(t, output.String(), "public")
					assert.Contains(t, output.String(), "[command]")
				} else {
					assert.NotContains(t, output.String(), "Commands:")
					assert.NotContains(t, output.String(), "[command]")
				}
				for _, command := range invocation.HelpInfo().Commands {
					assert.Equal(t, command.Name() == "public", command.IsVisible())
				}
				for _, name := range []string{"secret", "secret-alias"} {
					invocation := app.MustParse(context.Background(), nil, append(slices.Clone(path), name)...)
					require.NoError(t, invocation.Execute(nil))
					require.NoError(t, invocation.Close())
				}
				assert.Equal(t, 2, calls)
			})
		}
	}
}

func TestTemplatedHelpRequirementSnapshotsOverridesFunctionsAndEmptyTemplates(t *testing.T) {
	var output bytes.Buffer
	funcs := template.FuncMap{"title": strings.ToUpper}
	requirement := NewTemplatedHelpHandler().WithOutput(&output).
		WithApplicationTemplate(`{{title .FullName}}|{{template "usage" .}}{{range .Commands}}{{template "command-entry" .}}{{end}}{{range .Flags}}{{template "flag" .}}{{end}}`).
		WithCommandTemplate(`COMMAND:{{.FullName}}{{range .Arguments}}{{template "argument" .}}{{end}}`).
		WithUsageTemplate("USAGE:{{.Usage}}|").WithCommandEntryTemplate("CMD:{{.Label}}|").
		WithFlagTemplate("FLAG:{{.Label}}|").WithArgumentTemplate("ARG:{{.Name}}|").WithFuncs(funcs)
	handler, err := requirement.Create()
	require.NoError(t, err)
	requirement.ApplicationTemplate = "replacement"
	funcs["title"] = strings.ToLower
	requirement.SetOutput(io.Discard)
	app := NewSimpleApplication("app").
		WithHelpHandler(handler).
		WithCommand(NewSimpleCommand("child").
			WithArgument(NewArgument(codec.NewString(), "value")),
		).
		MustCreate()
	invocation := app.MustParse(context.Background(), nil, "--help")
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	assert.Equal(t, "APP|USAGE:app [flags] [command]|CMD:child|FLAG:-h, --help|", output.String())
	output.Reset()
	invocation, err = app.Parse(context.Background(), nil, "child", "--help")
	require.NoError(t, err)
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	assert.Equal(t, "COMMAND:app childARG:value|", output.String())
	output.Reset()
	empty, err := testDefaultTemplatedHelpRequirement().WithOutput(&output).WithApplicationTemplate("").Create()
	require.NoError(t, err)
	app, err = NewSimpleApplication("app").
		WithHelpHandler(empty).
		Create()
	require.NoError(t, err)
	invocation, err = app.Parse(context.Background(), nil, "--help")
	require.NoError(t, err)
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	assert.Empty(t, output.String())
}

type helpTestWriter struct {
	err    error
	short  bool
	cancel context.CancelFunc
}

func (this helpTestWriter) Write(value []byte) (int, error) {
	if this.cancel != nil {
		this.cancel()
	}
	if this.err != nil {
		return 0, this.err
	}
	if this.short {
		return 0, nil
	}
	return len(value), nil
}

func TestTemplatedHelpReportsConfigurationRenderWriterAndCancellationErrors(t *testing.T) {
	_, err := NewTemplatedHelpHandler().
		WithFlagTemplate("{{").
		Create()
	require.ErrorContains(t, err, "help template flag")
	_, err = NewTemplatedHelpHandler().
		WithFuncs(template.FuncMap{"bad": 42}).
		Create()
	require.ErrorContains(t, err, "help template functions")
	var output bytes.Buffer
	handler, err := NewTemplatedHelpHandler().
		WithOutput(&output).
		WithApplicationTemplate("partial{{.MissingField}}").
		Create()
	require.NoError(t, err)
	app, err := NewSimpleApplication("app").
		WithHelpHandler(handler).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.ErrorContains(t, invocation.Execute(nil), "render application help")
	assert.Empty(t, output.String(), "render failures must not write partial output")
	writeFailure := errors.New("writer failed")
	for _, writer := range []helpTestWriter{{err: writeFailure}, {short: true}} {
		handler, err := NewTemplatedHelpHandler().WithOutput(writer).WithApplicationTemplate("help").Create()
		require.NoError(t, err)
		err = handler(context.Background(), invocation)
		if writer.short {
			require.ErrorIs(t, err, io.ErrShortWrite)
		} else {
			require.ErrorIs(t, err, writeFailure)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	handler, err = NewTemplatedHelpHandler().WithOutput(helpTestWriter{cancel: cancel}).WithApplicationTemplate("help").Create()
	require.NoError(t, err)
	require.ErrorIs(t, handler(ctx, invocation), context.Canceled)
	require.ErrorContains(t, handler(context.Background(), nil), "requires an invocation")
	var typedNil *Invocation[NoOptions]
	require.ErrorContains(t, handler(context.Background(), typedNil), "requires an invocation")
}

func TestTemplatedHelpSerializesConcurrentWritesToItsConfiguredWriter(t *testing.T) {
	var output bytes.Buffer
	handler, err := NewTemplatedHelpHandler().WithOutput(&output).WithApplicationTemplate("{{.FullName}}\n").Create()
	require.NoError(t, err)
	app, err := NewSimpleApplication("app").
		WithHelpHandler(handler).
		Create()
	require.NoError(t, err)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(context.Background(), nil, "--help")
			if !assert.NoError(t, err) {
				return
			}
			assert.NoError(t, invocation.Execute(nil))
			assert.NoError(t, invocation.Close())
		})
	}
	workers.Wait()
	assert.Equal(t, strings.Repeat("app\n", 32), output.String())
}

func TestTemplatedHelpWithoutConfiguredTemplatesIsIntentionallyEmpty(t *testing.T) {
	for _, requirement := range []*TemplatedHelpHandlerRequirement{
		{},
		NewTemplatedHelpHandler(),
	} {
		var output bytes.Buffer
		requirement.SetOutput(&output)
		handler, err := requirement.Create()
		require.NoError(t, err)
		app := NewSimpleApplication("app").
			WithHelpHandler(handler).
			WithCommand(NewSimpleCommand("child")).
			MustCreate()
		for _, args := range [][]string{{"--help"}, {"child", "--help"}} {
			invocation := app.MustParse(context.Background(), nil, args...)
			require.NoError(t, invocation.Execute(nil))
			require.NoError(t, invocation.Close())
			assert.Empty(t, output.String())
		}
	}
}

func TestHelpHandlersAreCapturedAndHelpDisabledIsCapturedAtCreate(t *testing.T) {
	var output bytes.Buffer
	first := NewTemplatedHelpHandler().
		WithOutput(&output).
		WithApplicationTemplate("original").
		MustCreate()
	second := NewTemplatedHelpHandler().
		WithOutput(&output).
		WithApplicationTemplate("replacement").
		MustCreate()
	requirement := NewSimpleApplication("app").
		WithHelpHandler(first)
	app, err := requirement.Create()
	require.NoError(t, err)
	requirement.SetHelp(false)
	requirement.SetHelpHandler(second)
	invocation, err := app.Parse(context.Background(), nil, "--help")
	require.NoError(t, err)
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	assert.Equal(t, "original", output.String())
}

func TestDefaultHelpHandlerOverrideIsCapturedAndExplicitHandlerTakesPrecedence(t *testing.T) {
	original := DefaultHelpHandler
	defer func() { DefaultHelpHandler = original }()
	var calls []string
	DefaultHelpHandler = func(context.Context, InvocationLike) error {
		calls = append(calls, "first default")
		return nil
	}
	first, err := NewSimpleApplication("first").
		Create()
	require.NoError(t, err)
	DefaultHelpHandler = func(context.Context, InvocationLike) error {
		calls = append(calls, "second default")
		return nil
	}
	second, err := NewSimpleApplication("second").
		Create()
	require.NoError(t, err)
	explicit, err := NewSimpleApplication("explicit").
		WithHelpHandler(func(context.Context, InvocationLike) error {
			calls = append(calls, "explicit")
			return nil
		}).
		Create()
	require.NoError(t, err)
	for _, app := range []*Application[NoOptions]{first, second, explicit} {
		invocation, err := app.Parse(context.Background(), nil, "--help")
		require.NoError(t, err)
		require.NoError(t, invocation.Execute(nil))
		require.NoError(t, invocation.Close())
	}
	assert.Equal(t, []string{"first default", "second default", "explicit"}, calls)
	DefaultHelpHandler = nil
	_, err = NewSimpleApplication("app").
		Create()
	require.ErrorContains(t, err, "DefaultHelpHandler is nil")
	_, err = NewSimpleApplication("app").
		WithHelp(false).
		Create()
	require.NoError(t, err)
	_, err = NewSimpleApplication("app").
		WithHelpHandler(func(context.Context, InvocationLike) error {
			return nil
		}).
		Create()
	require.NoError(t, err)
}

// The helper runs only in a subprocess. A nonzero configured code must really
// terminate the process; errors must return without reaching os.Exit.
func TestTemplatedHelpExitCodeProcess(t *testing.T) {
	mode := os.Getenv("BIFROEST_CLI_HELP_EXIT_TEST")
	if mode != "" {
		requirement := NewTemplatedHelpHandler().WithExitCode(7).WithApplicationTemplate("HELP-OUTPUT\n")
		switch mode {
		case "zero":
			requirement.SetExitCode(0)
		case "writer error":
			requirement.SetOutput(helpTestWriter{err: errors.New("writer failed")})
		case "render error":
			requirement.SetApplicationTemplate("partial{{.MissingField}}")
		case "cancel":
			requirement.SetOutput(io.Discard)
		case "default":
			requirement.SetExitCode(0)
		}
		handler, err := requirement.Create()
		require.NoError(t, err)
		if mode == "snapshot" {
			requirement.SetExitCode(9)
		}
		if mode == "default" {
			handler = nil // Application must select the package DefaultHelpHandler
		}
		app, err := NewSimpleApplication("app").
			WithHelpHandler(handler).
			Create()
		require.NoError(t, err)
		invocation, err := app.Parse(context.Background(), nil, "--help")
		require.NoError(t, err)
		ctx := context.Background()
		if mode == "cancel" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		err = invocation.Execute(ctx)
		if mode == "writer error" || mode == "render error" || mode == "cancel" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.NoError(t, invocation.Close())
		return
	}
	for _, mode := range []string{"exit", "snapshot", "zero", "writer error", "render error", "cancel", "default"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestTemplatedHelpExitCodeProcess$")
			command.Env = append(os.Environ(), "BIFROEST_CLI_HELP_EXIT_TEST="+mode)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			if mode == "exit" || mode == "snapshot" {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				assert.Equal(t, 7, exit.ExitCode())
			} else {
				require.NoError(t, err, "stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
			if mode == "exit" || mode == "snapshot" || mode == "zero" {
				assert.Equal(t, "HELP-OUTPUT\n", stderr.String())
			}
			if mode == "default" {
				assert.Contains(t, stderr.String(), "Usage:")
				assert.Contains(t, stderr.String(), "-h, --help")
			}
			assert.NotContains(t, stdout.String(), "HELP-OUTPUT")
			assert.NotContains(t, stdout.String(), "Usage:")
		})
	}
}
