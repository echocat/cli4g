package cli

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type helpUnsafeOptions struct {
	Required string `flag:"required" env:"HELP_REQUIRED" required:"true"`
	Default  int    `flag:"default" default:"invalid"`
}

func (*helpUnsafeOptions) Validate(context.Context) error   { panic("help must not validate") }
func (*helpUnsafeOptions) Initialize(context.Context) error { panic("help must not initialize") }
func (*helpUnsafeOptions) Close() error                     { panic("help must not close uncreated options") }

func TestHelpInvocationSkipsOptionsDecodingAndLifecycleAndDispatchesToApplication(t *testing.T) {
	type childOptions struct {
		Name string `argument:"name" required:"true"`
	}
	var events []string
	var codecCalls atomic.Int64
	failure := errors.New("help handler failed")
	app := NewApplication[*helpUnsafeOptions]().
		WithName("app").
		WithCodec(callbackCodec[int, int]{
			Codec:   codec.NewInt(),
			onClone: func(context.Context) { codecCalls.Add(1) },
			onParse: func(context.Context) { codecCalls.Add(1) },
		}).
		WithCreateOptionsHandler(func(context.Context) (*helpUnsafeOptions, error) {
			events = append(events, "factory")
			return nil, errors.New("factory cannot run for help")
		}).
		WithValidateHandler(func(context.Context, *helpUnsafeOptions) error {
			events = append(events, "validate")
			return nil
		}).
		WithInitializeHandler(func(context.Context, *helpUnsafeOptions) error {
			events = append(events, "initialize")
			return nil
		}).
		WithHandler(func(context.Context, *helpUnsafeOptions) error {
			events = append(events, "execute")
			return nil
		}).
		WithCloseHandler(func(*helpUnsafeOptions) error {
			events = append(events, "close")
			return nil
		}).
		WithCommand(NewCommand[childOptions]().
			WithName("run", "r").
			WithCreateOptionsHandler(func(context.Context) (childOptions, error) {
				events = append(events, "child factory")
				return childOptions{}, nil
			}),
		).
		WithHelpHandler(func(ctx context.Context, invocation InvocationLike) error {
			fromContext, found := InvocationFromContext(ctx)
			assert.True(t, found)
			assert.Same(t, invocation, fromContext)
			container, found := ContainerFromContext(ctx)
			assert.True(t, found)
			assert.Equal(t, "run", container.Name())
			assert.Equal(t, []string{"run"}, invocation.NameStack())
			events = append(events, "help")
			return failure
		}).
		MustCreate()
	invocation := app.MustParse(nil, &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
		assert.Fail(t, "help must not look up environment values")
		return "", false
	}}, "--default=also-invalid", "r", "--help", "--unknown")
	defer func() { assert.NoError(t, invocation.Close()) }()
	assert.True(t, invocation.HelpRequested())
	_, options := invocation.Application()
	assert.Nil(t, options)
	stack := invocation.Stack()
	require.Len(t, stack, 1)
	assert.Nil(t, stack[0].Options)
	command, found := FindCommand[childOptions](app, "run")
	require.True(t, found)
	_, found = command.OptionsFrom(invocation)
	assert.False(t, found)
	for range invocation.Flags() {
		assert.Fail(t, "help has no decoded flags")
	}
	for range invocation.Arguments() {
		assert.Fail(t, "help has no decoded arguments")
	}
	assert.Empty(t, events)
	assert.Zero(t, codecCalls.Load())
	require.ErrorIs(t, invocation.Execute(nil), failure)
	assert.Equal(t, []string{"help"}, events)
	require.NoError(t, invocation.Close())
	require.NoError(t, invocation.Close())
	assert.ErrorIs(t, invocation.Execute(nil), ErrInvocationClosed)
	assert.Equal(t, []string{"run"}, invocation.NameStack())
	assert.Equal(t, []string{"help"}, events)
}

func TestHelpScannerStopsAtCurrentContainerAndRespectsValueAndTerminatorGrammar(t *testing.T) {
	type options struct {
		Name    string   `flag:"name"`
		Number  int      `flag:"number"`
		Verbose bool     `flag:"verbose" short:"v"`
		Values  []string `argument:"values"`
	}
	app := NewApplication[options]().
		WithName("app").
		WithCommand(NewSimpleCommand().
			WithName("run", "r"),
		).
		MustCreate()
	for _, test := range []struct {
		name string
		args []string
		help bool
		path []string
		err  string
	}{
		{name: "long root", args: []string{"--help"}, help: true},
		{name: "short root", args: []string{"-h"}, help: true},
		{name: "explicit true", args: []string{"--help=true"}, help: true},
		{name: "leaf", args: []string{"r", "--help"}, help: true, path: []string{"run"}},
		{name: "stop before command", args: []string{"--help", "run"}, help: true},
		{name: "ignore trailing error", args: []string{"run", "--help", "--unknown"}, help: true, path: []string{"run"}},
		{name: "ignore undecoded invalid value", args: []string{"--number=invalid", "run", "--help"}, help: true, path: []string{"run"}},
		{name: "short cluster stops", args: []string{"-vhz"}, help: true},
		{name: "earlier long error", args: []string{"--unknown", "--help"}, err: "unknown flag --unknown"},
		{name: "earlier short error", args: []string{"-zh"}, err: "unknown short flag -z"},
		{name: "bad bool", args: []string{"--help=invalid"}, err: "flag --help"},
		{name: "false continues", args: []string{"--help=false"}},
		{name: "false then true", args: []string{"--help=false", "--help"}, help: true},
		{name: "required flag value", args: []string{"--name", "--help"}},
		{name: "terminator", args: []string{"--", "--help"}},
		{name: "command selection already ended", args: []string{"value", "run", "--help"}, help: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(context.Background(), nil, test.args...)
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				assert.Nil(t, invocation)
				return
			}
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			assert.Equal(t, test.help, invocation.HelpRequested())
			if len(test.path) == 0 {
				assert.Empty(t, invocation.NameStack())
			} else {
				assert.Equal(t, test.path, invocation.NameStack())
			}
			if test.name == "required flag value" {
				_, value := invocation.Application()
				assert.Equal(t, "--help", value.Name)
			}
			if test.name == "terminator" {
				_, value := invocation.Application()
				assert.Equal(t, []string{"--help"}, value.Values)
			}
		})
	}
}

func TestHelpCanBeDisabledAndHelpNamesBecomeOrdinaryFlags(t *testing.T) {
	requirement := NewSimpleApplication().WithName("app")
	requirement.SetHelp(false)
	app := requirement.WithFlag(NewFlag(codec.NewString()).
		WithName("help").
		WithShort('h')).
		WithCommand(NewSimpleCommand().
			WithName("run"),
		).
		MustCreate()
	for _, args := range [][]string{{"--help=value"}, {"run", "-hvalue"}} {
		invocation := app.MustParse(context.Background(), nil, args...)
		require.NoError(t, invocation.Close())
		assert.False(t, invocation.HelpRequested())
		_, value := invocation.Flag("help")
		assert.Equal(t, "value", value)
	}
	app = NewSimpleApplication().
		WithName("app").
		WithHelp(false).
		MustCreate()
	_, err := app.Parse(context.Background(), nil, "--help")
	require.ErrorContains(t, err, "unknown flag --help")
}

func TestAutomaticHelpRejectsNamesShortsAliasesAndTagsAcrossTheTree(t *testing.T) {
	for _, flag := range []*FlagRequirement[string, string]{
		NewFlag(codec.NewString()).
			WithName("help"),
		NewFlag(codec.NewString()).
			WithName("value", "help"),
		NewFlag(codec.NewString()).
			WithName("value").
			WithShort('h'),
	} {
		_, err := NewSimpleApplication().
			WithName("app").
			WithFlag(flag).
			Create()
		require.Error(t, err)
		_, err = NewSimpleApplication().
			WithName("app").
			WithCommand(NewSimpleCommand().
				WithName("child").
				WithFlag(flag),
			).Create()
		require.Error(t, err)
	}
	type options struct {
		Value string `flag:"value|help"`
	}
	_, err := NewApplication[options]().WithName("app").Create()
	require.ErrorContains(t, err, "help")
	type shortOptions struct {
		Value string `flag:"value" short:"h"`
	}
	_, err = NewSimpleApplication().
		WithName("app").
		WithCommand(NewCommand[shortOptions]().
			WithName("child"),
		).Create()
	require.ErrorContains(t, err, "h")
}

func TestHelpMetadataUsesSelectedCodecOverridesAndReturnsIndependentLists(t *testing.T) {
	type options struct {
		Value string `flag:"value" default:"configured"`
	}
	app := NewApplication[options]().WithName("app").
		WithCodec(grammarString{min: 1, max: 1, prefix: "root:"}).
		WithCommand(NewSimpleCommand().
			WithName("child").
			WithCodec(grammarString{min: 1, max: 1, prefix: "child:"}),
		).
		MustCreate()
	invocation := app.MustParse(context.Background(), nil, "child", "--help")
	defer func() { assert.NoError(t, invocation.Close()) }()
	info := invocation.HelpInfo()
	assert.Equal(t, "child", info.Target.Name())
	require.Len(t, info.Path, 2)
	require.Len(t, info.Flags, 2)
	assert.True(t, info.Flags[0].Inherited)
	assert.Same(t, app, info.Flags[0].Owner)
	formatted, _, err := info.Flags[0].Definition.FormatDefault(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"child:configured"}, formatted)
	rootFlag, found := app.Flag("value")
	require.True(t, found)
	formatted, _, err = rootFlag.FormatDefault(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"root:configured"}, formatted)
	info.Path[0] = nil
	info.Flags[0] = HelpFlagInfo{}
	assert.Same(t, app, invocation.HelpInfo().Path[0])
	assert.NotNil(t, invocation.HelpInfo().Flags[0].Definition)
}

func TestHelpExecuteCancellationAndConcurrentLiveInvocations(t *testing.T) {
	var calls atomic.Int64
	app, err := NewSimpleApplication().WithName("app").WithHelpHandler(func(context.Context, InvocationLike) error {
		calls.Add(1)
		return nil
	}).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, invocation.Execute(ctx), context.Canceled)
	assert.Zero(t, calls.Load())
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			other, err := app.Parse(context.Background(), nil, "-h")
			if !assert.NoError(t, err) {
				return
			}
			assert.NoError(t, other.Execute(nil))
			assert.NoError(t, other.Close())
		})
	}
	workers.Wait()
	assert.EqualValues(t, 32, calls.Load())
	assert.NoError(t, invocation.Execute(nil))
	assert.EqualValues(t, 33, calls.Load())
}
