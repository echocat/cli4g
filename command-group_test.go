package cli

import (
	"bytes"
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestCommandGroupRequiresSubcommandBeforeOptionsLifecycle(t *testing.T) {
	for _, test := range []struct {
		name          string
		args          []string
		path          string
		parentHandler bool
		helpEnabled   bool
	}{
		{name: "application", path: "app", helpEnabled: true},
		{name: "nested alias", args: []string{"g"}, path: "app group", helpEnabled: true},
		{name: "ancestor handler", args: []string{"group"}, path: "app group", parentHandler: true, helpEnabled: true},
		{name: "help flag disabled", args: []string{"group"}, path: "app group"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			factory := func(context.Context) (NoOptions, error) { calls++; return NoOptions{}, nil }
			hook := func(context.Context, NoOptions) error { calls++; return nil }
			appRequirement := NewSimpleApplication("app").WithHelp(test.helpEnabled).
				WithCreateOptionsHandler(factory).WithValidateHandler(hook).WithInitializeHandler(hook).
				WithFlag(NewFlag(codec.NewString(), "required").WithEnvVar("REQUIRED").WithRequired(true),
					NewFlag(codec.NewInt32(), "default").WithDefaultStr("invalid")).
				WithCommand(NewSimpleCommand("group").WithAliases("g").
					WithCreateOptionsHandler(factory).WithValidateHandler(hook).WithInitializeHandler(hook).
					WithCommand(NewSimpleCommand("leaf").WithHandler(hook)))
			if test.parentHandler {
				appRequirement.WithHandler(hook)
			}
			app := appRequirement.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			lookupCalls := 0
			invocation, err := app.Parse(nil, &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				lookupCalls++
				return "", false
			}}, test.args...)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, ErrValidation)
			assert.EqualError(t, err, `a subcommand is required for "`+test.path+`"`)
			assert.Zero(t, calls)
			assert.Zero(t, lookupCalls)
		})
	}
}

func TestCommandGroupErrorRendersSelectedHelp(t *testing.T) {
	var output bytes.Buffer
	help := testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()
	response := NewRespondErrorHandler().WithOutput(&output).WithHelpHandler(help).MustCreate()
	reports := 0
	app := NewSimpleApplication("app").WithCommand(NewSimpleCommand("group").WithAliases("g").
		WithCommand(NewSimpleCommand("visible", "Visible command.").WithProfiles("A"),
			NewSimpleCommand("inactive", "Inactive command.").WithProfiles("B"))).
		WithHelpHandler(help).WithErrorHandler(func(ctx context.Context, invocation InvocationLike, err error) error {
		reports++
		current, found := ContainerFromContext(ctx)
		require.True(t, found)
		assert.Equal(t, "group", current.Name())
		assert.Equal(t, []string{"group"}, invocation.NameStack())
		assert.Equal(t, []string{"A"}, slices.Collect(invocation.Profiles()))
		assert.False(t, invocation.HelpRequested())
		assert.Nil(t, invocation.Stack()[0].Options)
		return response(ctx, invocation, err)
	}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	err := app.ParseAndExecute(nil, &ParseOpts{Profiles: []string{"A"}}, "g")
	require.ErrorIs(t, err, ErrValidation)
	assert.Equal(t, 1, reports)
	assert.Contains(t, output.String(), `Error: a subcommand is required for "app group"`)
	assert.Contains(t, output.String(), "Usage:\n  app group")
	assert.Contains(t, output.String(), "Visible command.")
	assert.NotContains(t, output.String(), "Inactive command.")
}

func TestCommandGroupExplicitHelpSkipsLifecycle(t *testing.T) {
	calls, helpCalls := 0, 0
	app := NewSimpleApplication("app").
		WithCreateOptionsHandler(func(context.Context) (NoOptions, error) { calls++; return NoOptions{}, nil }).
		WithHandler(func(context.Context, NoOptions) error { calls++; return nil }).
		WithCommand(NewSimpleCommand("group").WithCommand(NewSimpleCommand("leaf"))).
		WithHelpHandler(func(_ context.Context, invocation InvocationLike) error {
			helpCalls++
			assert.Equal(t, "group", invocation.HelpInfo().Target.Name())
			return nil
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, flag := range []string{"--help", "-h"} {
		assert.NoError(t, app.ParseAndExecute(nil, nil, "group", flag))
	}
	assert.Zero(t, calls)
	assert.Equal(t, 2, helpCalls)
	_, err := app.Parse(nil, nil, "group", "--unknown")
	assert.EqualError(t, err, "unknown flag --unknown", "scanner errors retain precedence")
}

func TestCommandGroupOwnHandlerAllowsDirectInvocation(t *testing.T) {
	var calls []string
	app := NewSimpleApplication("app").WithHandler(func(context.Context, NoOptions) error {
		calls = append(calls, "app")
		return nil
	}).WithCommand(NewSimpleCommand("group").WithHandler(func(context.Context, NoOptions) error {
		calls = append(calls, "group")
		return nil
	}).WithCommand(NewSimpleCommand("leaf"))).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.NoError(t, app.ParseAndExecute(nil, nil))
	assert.NoError(t, app.ParseAndExecute(nil, nil, "group"))
	assert.Equal(t, []string{"app", "app", "group"}, calls)
}

func TestCommandGroupOnlyActiveChildrenRequireSelection(t *testing.T) {
	app := NewSimpleApplication("app").WithCommand(NewSimpleCommand("group").
		WithCommand(NewSimpleCommand("child").WithProfiles("A"))).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.NoError(t, app.ParseAndExecute(nil, nil, "group"), "no active children permits parse-only use")
	_, err := app.Parse(nil, &ParseOpts{Profiles: []string{"A"}}, "group")
	assert.EqualError(t, err, `a subcommand is required for "app group"`)
	assert.NoError(t, app.ParseAndExecute(nil, &ParseOpts{Profiles: []string{"A"}}, "group", "child"))
}
