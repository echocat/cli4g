package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestConstructorsNameAndDescription(t *testing.T) {
	for _, test := range []struct {
		name        string
		values      []string
		wantName    string
		description string
	}{
		{name: "unconfigured"},
		{name: "name only", values: []string{"example"}, wantName: "example"},
		{name: "name and description", values: []string{"example", "Example description."}, wantName: "example", description: "Example description."},
		{name: "multiple description parts", values: []string{"example", "An example", "with several", "parts."}, wantName: "example", description: "An example with several parts."},
		{name: "empty name", values: []string{"", "description"}, description: "description"},
		{name: "preserve whitespace", values: []string{"example", " leading ", "", " trailing "}, wantName: "example", description: " leading    trailing "},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApplication[NoOptions](test.values...)
			assert.Equal(t, test.wantName, app.Name)
			assert.Equal(t, test.description, app.Description)
			assert.True(t, app.Help)
			assert.True(t, app.FallbackToStdio)
			simpleApp := NewSimpleApplication(test.values...)
			assert.Equal(t, app.Name, simpleApp.Name)
			assert.Equal(t, app.Description, simpleApp.Description)
			assert.True(t, simpleApp.Help)
			assert.True(t, simpleApp.FallbackToStdio)

			var names []string
			if len(test.values) > 0 {
				names = []string{test.wantName}
			}
			command := NewCommand[NoOptions](test.values...)
			assert.Equal(t, names, command.Names)
			assert.Equal(t, test.description, command.Description)
			assert.True(t, command.Visible)
			simpleCommand := NewSimpleCommand(test.values...)
			assert.Equal(t, names, simpleCommand.Names)
			assert.Equal(t, test.description, simpleCommand.Description)
			assert.True(t, simpleCommand.Visible)
			decoder := codec.NewString()
			flag := NewFlag(decoder, test.values...)
			assert.Equal(t, names, flag.Names)
			assert.Equal(t, test.description, flag.Description)
			assert.Equal(t, decoder, flag.Codec)
			assert.True(t, flag.Visible)
			argument := NewArgument(decoder, test.values...)
			assert.Equal(t, test.wantName, argument.Name)
			assert.Equal(t, test.description, argument.Description)
			assert.Equal(t, decoder, argument.Codec)
			if len(test.values) > 0 {
				test.values[0] = "mutated"
				assert.Equal(t, names, command.Names, "constructor must not retain the caller's slice")
				assert.Equal(t, names, flag.Names)
			}
		})
	}
}

func TestConstructorMetadataCompilesAndCanBeOverridden(t *testing.T) {
	flag := NewFlag(codec.NewString(), "original", "Original description.").
		WithName("message").WithAliases("m").WithDescription("Message description.")
	argument := NewArgument(codec.NewInt32(), "original", "Original description.").
		WithName("count").WithDescription("Count description.")
	command := NewSimpleCommand("original", "Original description.").
		WithName("run").WithAliases("r").WithDescription("Run description.").WithFlag(flag).WithArgument(argument)
	app, err := NewSimpleApplication("original", "Original description.").
		WithName("app").WithDescription("Application description.").WithCommand(command).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "r", "--m", "hello", "42")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	assert.Equal(t, "app", app.Name())
	assert.Equal(t, "Application description.", app.Description())
	assert.Equal(t, []string{"run"}, invocation.NameStack())
	definition, value := invocation.Flag("message")
	require.NotNil(t, definition)
	assert.Equal(t, "hello", value)
	assert.Equal(t, []string{"message", "m"}, definition.Names())
	assert.Equal(t, "Message description.", definition.Description())
	position, value := invocation.Argument(0)
	require.NotNil(t, position)
	assert.Equal(t, int32(42), value)
	assert.Equal(t, "count", position.Name())
	assert.Equal(t, "Count description.", position.Description())
	assert.Equal(t, "Run description.", invocation.HelpInfo().Target.Description())
}

func TestUnnamedConstructorsRemainConfigurableAndValidateAtCreate(t *testing.T) {
	_, err := NewSimpleApplication().Create()
	assert.ErrorContains(t, err, "application needs a name")
	_, err = NewSimpleApplication("app").WithCommand(NewSimpleCommand()).Create()
	assert.ErrorContains(t, err, "command needs a name")
	_, err = NewSimpleApplication("app").WithFlag(NewFlag(codec.NewString())).Create()
	assert.ErrorContains(t, err, "flag needs a name")
	_, err = NewSimpleApplication("app").WithArgument(NewArgument(codec.NewString())).Create()
	assert.ErrorContains(t, err, "argument needs a name")
	// Existing setter-based construction remains supported for delayed configuration.
	app, err := NewSimpleApplication().WithName("app").WithCommand(NewSimpleCommand().WithName("run")).Create()
	require.NoError(t, err)
	assert.NoError(t, app.Close())
}
