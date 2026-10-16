package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestProfilesMetadataAndTypedVariants(t *testing.T) {
	type options struct {
		Text   string `flag:"foo|alias" short:"f" profile:"A" default:"hello"`
		Number int32  `flag:"foo|alias" short:"f" profile:"B"`
		Shared string `flag:"shared" profile:"A|B" default:"shared"`
		Global bool   `flag:"global" short:"g"`
	}
	app := NewApplication[options]("app").
		WithCommand(NewSimpleCommand("group").
			WithProfiles("C").
			WithCommand(NewSimpleCommand("nested").
				WithProfiles("D"),
			),
		).
		MustCreate()
	t.Cleanup(func() { assert.NoError(t, app.Close()) })
	assert.Equal(t, []string{"A", "B", "C", "D"}, slices.Collect(app.Profiles()))
	assert.True(t, app.ContainsProfile("A"))
	assert.False(t, app.ContainsProfile("a"))
	assert.False(t, app.ContainsProfile(""))
	assert.Len(t, slices.Collect(app.Flags()), 5) // includes automatic help
	_, found := app.Flag("foo")
	assert.False(t, found)
	_, found = app.ShortFlag('f')
	assert.False(t, found)
	a, found := app.FlagOfProfile("alias", "A")
	require.True(t, found)
	b, found := app.ShortFlagOfProfile('f', "B")
	require.True(t, found)
	assert.NotEqual(t, a.ValueType(), b.ValueType())
	assert.True(t, a.HasDefault())
	assert.False(t, b.HasDefault())
	assignments := a.Profiles()
	assignments[0] = "changed"
	assert.Equal(t, []string{"A"}, a.Profiles())
	_, found = app.FlagOfProfile("global", "A")
	assert.False(t, found)
	_, found = app.FlagOfProfile("foo", "unknown")
	assert.False(t, found)
	_, found = app.Command("group")
	assert.False(t, found)
	group, found := app.CommandOfProfile("group", "C")
	require.True(t, found)
	assert.Equal(t, []string{"C"}, group.Profiles())
	for _, test := range []struct {
		profile string
		args    []string
		text    string
		number  int32
	}{
		{profile: "A", text: "hello"},
		{profile: "A", args: []string{"--alias=world"}, text: "world"},
		{profile: "B"},
		{profile: "B", args: []string{"-f42"}, number: 42},
	} {
		t.Run(fmt.Sprint(test.profile, test.args), func(t *testing.T) {
			invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{test.profile, test.profile}}, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, actual := invocation.Application()
			assert.Equal(t, options{Text: test.text, Number: test.number, Shared: "shared"}, actual)
		})
	}
	invocation, err := app.Parse(nil, nil, "--global")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, actual := invocation.Application()
	assert.Equal(t, options{Global: true}, actual)
	_, err = app.Parse(nil, nil, "--foo=unavailable")
	assert.ErrorContains(t, err, "unknown flag --foo")
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"A", "B"}}, "--help")
	assert.ErrorContains(t, err, `flag "foo" is duplicated`)
	assert.ErrorContains(t, err, "selected profiles [A B]")
}

func TestProfilesCreateRejectsStaticallyOverlappingDefinitions(t *testing.T) {
	for _, test := range []struct {
		name string
		a, b []string
	}{
		{name: "both global"},
		{name: "global then profile", b: []string{"A"}},
		{name: "profile then global", a: []string{"A"}},
		{name: "same profile", a: []string{"A"}, b: []string{"A"}},
		{name: "intersecting profiles", a: []string{"A", "B"}, b: []string{"B", "C"}},
	} {
		for _, kind := range []string{"name", "alias", "short", "argument", "command", "command alias", "inherited", "binding"} {
			t.Run(test.name+"/"+kind, func(t *testing.T) {
				type options struct{ Value string }
				r := NewApplication[options]("app")
				a := NewFlag(codec.NewString(), "foo").
					WithProfiles(test.a...)
				b := NewFlag(codec.NewString(), "foo").
					WithProfiles(test.b...)
				switch kind {
				case "name":
					r.WithFlag(a, b)
				case "alias":
					r.WithFlag(a.WithName("one", "alias"), b.WithName("two", "alias"))
				case "short":
					r.WithFlag(a.WithName("one").WithShort('x'), b.WithName("two").WithShort('x'))
				case "argument":
					r.WithArgument(NewArgument(codec.NewString(), "foo").WithProfiles(test.a...), NewArgument(codec.NewString(), "foo").WithProfiles(test.b...))
				case "command", "command alias":
					one := NewSimpleCommand("foo").
						WithProfiles(test.a...)
					two := NewSimpleCommand("foo").
						WithProfiles(test.b...)
					if kind == "command alias" {
						one.WithName("one", "alias")
						two.WithName("two", "alias")
					}
					r.WithCommand(one, two)
				case "inherited":
					r.
						WithFlag(a).
						WithCommand(NewSimpleCommand("child").
							WithFlag(b),
						)
				case "binding":
					r.WithFlag(
						a.WithName("one").
							WithField("Value"),
						b.WithName("two").
							WithField("Value"),
					)
				}
				_, err := r.Create()
				require.Error(t, err)
			})
		}
	}
	for _, profiles := range [][]string{{""}, {"A", ""}, {"A|B"}} {
		_, err := NewSimpleApplication("app").
			WithCommand(NewSimpleCommand("child").
				WithProfiles(profiles...),
			).
			Create()
		assert.ErrorContains(t, err, "invalid profile")
	}
	type invalid struct {
		Value int32 `flag:"foo" profile:"unused" default:"not-an-integer"`
	}
	// Default syntax is validated during parsing/help in the existing lifecycle;
	// structural errors must nevertheless be caught even for inactive definitions.
	_, err := NewApplication[invalid]("app").
		WithFlag(NewFlag(codec.NewString(), "other").
			WithProfiles("unused").
			WithField("Value"),
		).
		Create()
	assert.ErrorContains(t, err, "codec returns string")
}

func TestProfilesCombinationConflictsBeforeOptionsCreation(t *testing.T) {
	for _, kind := range []string{"alias", "short", "argument", "command", "inherited", "binding", "variadic", "required parent"} {
		t.Run(kind, func(t *testing.T) {
			type options struct{ Value string }
			created := 0
			r := NewApplication[options]("app").
				WithCreateOptionsHandler(func(context.Context) (options, error) {
					created++
					return options{}, nil
				})
			a := NewFlag(codec.NewString(), "one").
				WithProfiles("A")
			b := NewFlag(codec.NewString(), "two").
				WithProfiles("B")
			var args []string
			switch kind {
			case "alias":
				r.WithFlag(a.WithAliases("alias"), b.WithAliases("alias"))
			case "short":
				r.WithFlag(a.WithShort('x'), b.WithShort('x'))
			case "argument":
				r.WithArgument(
					NewArgument(codec.NewString(), "value").
						WithProfiles("A"),
					NewArgument(codec.NewString(), "value").
						WithProfiles("B"),
				)
			case "command":
				r.WithCommand(
					NewSimpleCommand("child").
						WithProfiles("A"),
					NewSimpleCommand("child").
						WithProfiles("B"),
				)
			case "inherited":
				r.WithFlag(a.WithName("foo")).
					WithCommand(NewSimpleCommand("child").
						WithFlag(b.WithName("foo")),
					)
				args = []string{"child"}
			case "binding":
				r.WithFlag(a.WithField("Value"), b.WithField("Value"))
			case "variadic":
				r.WithArgument(
					NewArgument(codec.NewStrings(), "many").
						WithProfiles("A"),
					NewArgument(codec.NewString(), "last").
						WithProfiles("B"),
				)
			case "required parent":
				r.
					WithArgument(NewArgument(codec.NewString(), "value").
						WithProfiles("A").
						WithRequired(true),
					).
					WithCommand(NewSimpleCommand("child").
						WithProfiles("B"),
					)
			}
			app, err := r.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A", "B"}}, args...)
			require.Error(t, err)
			assert.Nil(t, invocation)
			assert.True(t, IsValidationError(err))
			assert.ErrorContains(t, err, "selected profiles [A B]")
			assert.Zero(t, created)
		})
	}
}

func TestProfilesArgumentFilteringAndStableIndices(t *testing.T) {
	type options struct {
		A    string `argument:"value" profile:"A"`
		B    int32  `argument:"value" profile:"B"`
		Tail string `argument:"tail"`
	}
	app := NewApplication[options]("app").
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"B"}}, "42", "last")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, actual := invocation.Application()
	assert.Equal(t, options{B: 42, Tail: "last"}, actual)
	argument, _ := invocation.Argument(0)
	assert.Nil(t, argument)
	argument, value := invocation.Argument(1)
	require.NotNil(t, argument)
	assert.Equal(t, int32(42), value)
	definition, found := app.Argument(1)
	require.True(t, found)
	assert.Same(t, definition, argument)
	assert.Len(t, invocation.HelpInfo().Arguments, 2)
	global, err := app.Parse(nil, nil, "only-tail")
	require.NoError(t, err)
	defer func() { assert.NoError(t, global.Close()) }()
	_, actual = global.Application()
	assert.Equal(t, options{Tail: "only-tail"}, actual)
}

func TestProfilesCommandsAndAncestorGates(t *testing.T) {
	type textOptions struct {
		Value string `flag:"value" default:"hello"`
	}
	type numberOptions struct {
		Value int32 `flag:"value"`
	}
	app := NewSimpleApplication("app").
		WithCommand(
			NewCommand[textOptions]("convert").
				WithAliases("c").
				WithProfiles("A"),
			NewCommand[numberOptions]("convert").
				WithAliases("c").
				WithProfiles("B"),
			NewSimpleCommand("admin").
				WithProfiles("A").
				WithCommand(
					NewSimpleCommand("cleanup").
						WithProfiles("B"),
					NewSimpleCommand("status"),
				),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	a, found := FindCommandOfProfile[textOptions](app, "A", "c")
	require.True(t, found)
	b, found := FindCommandOfProfile[numberOptions](app, "B", "convert")
	require.True(t, found)
	_, found = FindCommand[textOptions](app, "convert")
	assert.False(t, found)
	for _, test := range []struct {
		profile string
		value   string
	}{{"A", "world"}, {"B", "42"}} {
		invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{test.profile}}, "c", "--value", test.value)
		require.NoError(t, err)
		if test.profile == "A" {
			actual, found := a.OptionsFrom(invocation)
			require.True(t, found)
			assert.Equal(t, "world", actual.Value)
			_, found = b.OptionsFrom(invocation)
			assert.False(t, found)
		} else {
			actual, found := b.OptionsFrom(invocation)
			require.True(t, found)
			assert.Equal(t, int32(42), actual.Value)
		}
		assert.NoError(t, invocation.Close())
	}
	// A separate tree avoids the intentionally conflicting convert siblings when
	// testing the additive ancestor/child profile gates.
	tree := NewSimpleApplication("tree").
		WithCommand(NewSimpleCommand("admin").
			WithProfiles("A").
			WithCommand(
				NewSimpleCommand("cleanup").
					WithProfiles("B"),
				NewSimpleCommand("status"),
			),
		).
		MustCreate()
	defer func() { assert.NoError(t, tree.Close()) }()
	for _, test := range []struct {
		profiles    []string
		args, names []string
	}{
		{profiles: []string{"A"}, args: []string{"admin", "status"}, names: []string{"admin", "status"}},
		{profiles: []string{"A", "B"}, args: []string{"admin", "cleanup"}, names: []string{"admin", "cleanup"}},
		{profiles: []string{"B"}, names: []string{}},
	} {
		invocation, err := tree.Parse(nil, &ParseOpts{Profiles: test.profiles}, test.args...)
		require.NoError(t, err)
		assert.Equal(t, test.names, invocation.NameStack())
		assert.NoError(t, invocation.Close())
	}
	_, err := tree.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "admin", "cleanup")
	assert.ErrorContains(t, err, "unexpected values")
}

func TestProfilesInactiveInputsHaveNoRuntimeEffects(t *testing.T) {
	type options struct {
		Output   *os.File
		Required string
	}
	name := filepath.Join(t.TempDir(), "must-not-exist")
	app := NewApplication[options]("app").
		WithFlag(
			NewFlag(codec.NewFile(codec.FileWrite), "output").
				WithProfiles("B").
				WithField("Output").
				WithDefaultStr(name),
			NewFlag(codec.NewString(), "required").
				WithProfiles("B").
				WithField("Required").
				WithEnvVar("REQUIRED").
				WithRequired(true),
			NewFlag(codec.NewBool(), "active").
				WithProfiles("A"),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	lookups := 0
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A"}, LookupEnv: func(context.Context, string) (string, bool) { lookups++; return "", false }})
	require.NoError(t, err)
	assert.NoError(t, invocation.Close())
	assert.Zero(t, lookups)
	_, err = os.Stat(name)
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"B"}, LookupEnv: func(context.Context, string) (string, bool) { return "", false }})
	assert.ErrorContains(t, err, "required flag --required is missing")
	_, err = os.Stat(name)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestProfilesHelpAndErrorViews(t *testing.T) {
	var output bytes.Buffer
	help := testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()
	created := 0
	var failureInfo HelpInfo
	app := NewSimpleApplication("app").
		WithHelpHandler(help).
		WithCreateOptionsHandler(func(context.Context) (NoOptions, error) { created++; return NoOptions{}, nil }).
		WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
			failureInfo = invocation.HelpInfo()
			return err
		}).
		WithFlag(
			NewFlag(codec.NewString(), "text-only").
				WithProfiles("A").
				WithDefault("hello"),
			NewFlag(codec.NewInt32(), "number-only").
				WithProfiles("B"),
		).
		WithCommand(
			NewSimpleCommand("a-only").
				WithProfiles("A"),
			NewSimpleCommand("b-only").
				WithProfiles("B"),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "--help", "--ignored")
	require.NoError(t, err)
	require.True(t, invocation.HelpRequested())
	assert.NoError(t, invocation.Execute(nil))
	assert.NoError(t, invocation.Close())
	assert.Contains(t, output.String(), "--text-only")
	assert.Contains(t, output.String(), "a-only")
	assert.NotContains(t, output.String(), "number-only")
	assert.NotContains(t, output.String(), "b-only")
	assert.Zero(t, created)
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"B"}}, "--unknown")
	require.Error(t, err)
	assert.Len(t, failureInfo.Flags, 2)
	assert.Equal(t, "number-only", failureInfo.Flags[0].Definition.Name())
	assert.Equal(t, "b-only", failureInfo.Commands[0].Name())
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"typo"}}, "--help")
	assert.ErrorContains(t, err, `unknown profile "typo"; known profiles: [A B]`)
	assert.Zero(t, created)
}

func TestProfilesConcurrentParses(t *testing.T) {
	type options struct {
		Text   string `flag:"foo" profile:"A" default:"hello"`
		Number int32  `flag:"foo" profile:"B"`
	}
	app := NewApplication[options]("app").
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var workers sync.WaitGroup
	for index := range 32 {
		workers.Go(func() {
			profile, value := "A", "world"
			if index%2 == 0 {
				profile, value = "B", "42"
			}
			invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{profile}}, "--foo", value)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, actual := invocation.Application()
			if profile == "A" {
				assert.Equal(t, options{Text: "world"}, actual)
			} else {
				assert.Equal(t, options{Number: 42}, actual)
			}
		})
	}
	workers.Wait()
}

func TestProfilesSharedAssignmentsAndSnapshot(t *testing.T) {
	assignments := []string{"A", "B"}
	flag := NewFlag(codec.NewString(), "shared").
		WithProfiles(assignments...).
		WithDefault("hello")
	command := NewSimpleCommand("shared-command").
		WithProfiles("A", "B")
	app := NewSimpleApplication("app").
		WithFlag(flag).
		WithCommand(command).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assignments[0] = "changed"
	flag.Profiles[0] = "changed-again"
	command.Profiles[0] = "changed-command"
	assert.Equal(t, []string{"A", "B"}, slices.Collect(app.Profiles()))
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A", "B", "A"}}, "shared-command")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("shared")
	assert.Equal(t, "hello", value)
	assert.Len(t, invocation.HelpInfo().Flags, 2) // shared once plus help
	assert.Len(t, slices.Collect(app.Commands()), 1)
	seen := 0
	for range app.Profiles() {
		seen++
		break
	}
	assert.Equal(t, 1, seen)
	for range app.Flags() {
		break
	}
	for range app.Commands() {
		break
	}
	empty := NewSimpleApplication("empty").MustCreate()
	defer func() { assert.NoError(t, empty.Close()) }()
	assert.Empty(t, slices.Collect(empty.Profiles()))
	_, err = empty.Parse(nil, &ParseOpts{Profiles: []string{"A"}})
	assert.ErrorContains(t, err, `unknown profile "A"`)
}

func TestProfilesSelectedCommandBoundariesAndArity(t *testing.T) {
	app := NewSimpleApplication("app").
		WithFlag(
			NewFlag(grammarString{min: 1, max: -1}, "many"),
			NewFlag(codec.NewBool(), "foo").
				WithShort('f').
				WithProfiles("A"),
			NewFlag(codec.NewString(), "foo").
				WithShort('f').
				WithProfiles("B"),
		).
		WithCommand(NewSimpleCommand("child").
			WithProfiles("B"),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		profile string
		args    []string
		many    string
		names   []string
		foo     any
	}{
		{profile: "A", args: []string{"-f", "--many", "one", "child"}, many: "one|child", names: []string{}, foo: true},
		{profile: "B", args: []string{"-fchild", "--many", "one", "child"}, many: "one", names: []string{"child"}, foo: "child"},
	} {
		invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{test.profile}}, test.args...)
		require.NoError(t, err)
		assert.Equal(t, test.names, invocation.NameStack())
		_, value := invocation.Flag("foo")
		assert.Equal(t, test.foo, value)
		_, value = invocation.Flag("many")
		assert.Equal(t, test.many, value)
		assert.NoError(t, invocation.Close())
	}
}

func TestProfilesUnvisitedSubtreeDoesNotConflict(t *testing.T) {
	app := NewSimpleApplication("app").
		WithCommand(
			NewSimpleCommand("safe"),
			NewSimpleCommand("conflicting").
				WithFlag(
					NewFlag(codec.NewString(), "foo").
						WithProfiles("A"),
					NewFlag(codec.NewInt32(), "foo").
						WithProfiles("B"),
				),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A", "B"}}, "safe", "--help")
	require.NoError(t, err)
	assert.True(t, invocation.HelpRequested())
	assert.NoError(t, invocation.Close())
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"A", "B"}}, "conflicting", "--help")
	assert.ErrorContains(t, err, "app conflicting: selected profiles [A B]")
}

func TestProfilesInheritedCodecOverridesSurviveHelpAndErrors(t *testing.T) {
	type options struct {
		Value string `flag:"value" profile:"A"`
	}
	rootCodec, err := codec.NewEnum(codec.NewString(), "root")
	require.NoError(t, err)
	childCodec, err := codec.NewEnum(codec.NewString(), "child")
	require.NoError(t, err)
	var output bytes.Buffer
	help := testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()
	var failureInfo HelpInfo
	app := NewApplication[options]("app").
		WithCodec(rootCodec).
		WithHelpHandler(help).
		WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
			failureInfo = invocation.HelpInfo()
			return err
		}).
		WithCommand(NewSimpleCommand("child").
			WithCodec(childCodec),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "--value", "child", "child")
	require.NoError(t, err)
	_, actual := invocation.Application()
	assert.Equal(t, "child", actual.Value)
	assert.NoError(t, invocation.Close())
	invocation, err = app.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "child", "--help")
	require.NoError(t, err)
	assert.NoError(t, invocation.Execute(nil))
	assert.NoError(t, invocation.Close())
	assert.Contains(t, output.String(), "choices: child")
	_, err = app.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "child", "--unknown")
	require.Error(t, err)
	definition := failureInfo.Flags[0].Definition
	_, err = definition.Codec().ParseRequirement(context.Background(), codec.UsageFlag, "child")
	assert.NoError(t, err)
}
