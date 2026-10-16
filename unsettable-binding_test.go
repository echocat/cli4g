package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type privateBindingMixin struct{ Value string }

func TestCreateRejectsUnexportedNonAnonymousBindingSteps(t *testing.T) {
	type options struct{ hidden struct{ Value string } } //nolint:unused // Deliberately unexported field inspected through reflection.
	_, err := NewApplication[options]("app").
		WithStringFlag("value", func(f *FlagRequirement[string, string]) { f.WithField("hidden", "Value") }).
		Create()
	require.ErrorContains(t, err, "hidden")
	require.ErrorContains(t, err, "exported")
}
func TestExportedLeafBehindPrivateAnonymousValueRemainsSettable(t *testing.T) {
	type options struct{ privateBindingMixin }
	app, err := NewApplication[*options]("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithStringFlag("value", func(f *FlagRequirement[string, string]) { f.WithField("Value") }).
		Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=assigned")
	require.NoError(t, err)
	defer func() { require.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, "assigned", value.Value)
}
