package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestHelpKeepsStrictInvalidDefaultBehavior(t *testing.T) {
	var output bytes.Buffer
	handler, err := testDefaultTemplatedHelpRequirement().WithOutput(&output).Create()
	require.NoError(t, err)
	app, err := NewSimpleApplication("app").
		WithErrorHandlerEnabled(false).
		WithHelpHandler(handler).
		WithIntFlag("number", func(f *FlagRequirement[int, int]) { f.WithDefaultStr("invalid") }).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	normal, err := app.Parse(nil, nil, "--number=7")
	require.NoError(t, err)
	require.NoError(t, normal.Close())
	help, err := app.Parse(nil, nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, help.Close()) }()
	err = help.Execute(nil)
	require.ErrorIs(t, err, codec.ErrIllegalArgument)
	assert.Zero(t, output.Len(), "strict data preparation failure prevents all help output")
}

type failingRequirementDisplay struct {
	codec.Codec[string, string]
	failure  error
	resolved *int
}

func (this failingRequirementDisplay) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (this failingRequirementDisplay) DisplayRequirement(context.Context, string) ([]string, error) {
	return nil, this.failure
}
func (this failingRequirementDisplay) Resolve(ctx context.Context, scope codec.Scope, value string) (string, error) {
	*this.resolved++
	return this.Codec.Resolve(ctx, scope, value)
}

func TestStrictHelpFormatFailureDoesNotResolveRuntimeValues(t *testing.T) {
	for _, kind := range []string{"flag", "argument"} {
		t.Run(kind, func(t *testing.T) {
			var output bytes.Buffer
			handler, err := testDefaultTemplatedHelpRequirement().WithOutput(&output).Create()
			require.NoError(t, err)
			resolved := 0
			failure := errors.New("display failed")
			original := "original"
			root := NewSimpleApplication("app").
				WithErrorHandlerEnabled(false).
				WithHelpHandler(handler)
			decoder := failingRequirementDisplay{Codec: codec.NewString(), failure: failure, resolved: &resolved}
			if kind == "flag" {
				root.WithFlag(NewFlag(decoder, "value").
					WithDefault(original),
				)
			} else {
				root.WithArgument(NewArgument(decoder, "value").
					WithDefault(original),
				)
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { _ = app.Close() }()
			invocation, err := app.Parse(nil, nil, "--help")
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			err = invocation.Execute(nil)
			require.ErrorIs(t, err, failure)
			assert.Zero(t, output.Len())
			assert.Zero(t, resolved)
			require.NoError(t, app.Close())
		})
	}
}
