package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpFlagValueName(t *testing.T) {
	for _, test := range []struct{ name, expected string }{
		{"name", "name"},
		{"fooBar", "bar"},
		{"log.colorMode", "mode"},
		{"log-mode", "mode"},
		{"log_mode", "mode"},
		{"serverURL", "url"},
		{"HTTPMode", "mode"},
		{"URL", "url"},
		{"log.HTTPMode", "mode"},
		{"maxRetryCount2", "count2"},
		{"http2Mode", "mode"},
		{"großeÄnderung", "änderung"},
		{"größe", "größe"},
		{"log..color_mode-", "mode"},
		{"._", "._"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, helpFlagValueName(test.name))
		})
	}
}

func TestHelpFlagLabelsUsePrimaryNameAndValueSyntax(t *testing.T) {
	for _, test := range []struct {
		name, placeholder, expected string
		min, max                    int
		aliases                     []string
		shorts                      []rune
	}{
		{name: "name", min: 1, max: 1, expected: "--name=<name>"},
		{name: "fooBar", min: 1, max: 1, expected: "--fooBar=<bar>"},
		{name: "log.colorMode", min: 1, max: 1, expected: "--log.colorMode=<mode>"},
		{name: "log-mode", min: 1, max: 1, expected: "--log-mode=<mode>"},
		{name: "log_mode", min: 1, max: 1, expected: "--log_mode=<mode>"},
		{name: "serviceAddress", aliases: []string{"remote-host"}, shorts: []rune{'a'}, min: 1, max: 1, expected: "-a, --serviceAddress, --remote-host=<address>"},
		{name: "log.colorMode", placeholder: "ColorMode", min: 1, max: 1, expected: "--log.colorMode=<ColorMode>"},
		{name: "log.colorMode", placeholder: "<ColorMode>", min: 1, max: 1, expected: "--log.colorMode=<ColorMode>"},
		{name: "mode", min: 0, max: 1, expected: "--mode[=<mode>]"},
		{name: "values", min: 1, max: -1, expected: "--values=<values>..."},
		{name: "values", min: 2, max: 3, expected: "--values=<values>{2..3}"},
		{name: "switch", min: 0, max: 0, expected: "--switch"},
	} {
		t.Run(test.expected, func(t *testing.T) {
			flag := NewFlag(grammarString{min: test.min, max: test.max}, test.name).
				WithAliases(test.aliases...).WithShort(test.shorts...).WithPlaceholder(test.placeholder)
			app := NewSimpleApplication("app").WithFlag(flag).MustCreate()
			t.Cleanup(func() { assert.NoError(t, app.Close()) })
			invocation := app.MustParse(t.Context(), nil, "--help")
			t.Cleanup(func() { assert.NoError(t, invocation.Close()) })
			data, err := buildHelpTemplateData(t.Context(), invocation)
			require.NoError(t, err)
			require.Len(t, data.Flags, 2)
			assert.Equal(t, test.expected, data.Flags[0].Label)
			assert.Equal(t, "-h, --help", data.Flags[1].Label)
		})
	}
}

func TestHelpArgumentLabelsPreserveNamesAndBracketUsageOnce(t *testing.T) {
	for _, test := range []struct {
		name, placeholder, label, usage string
		min, max                        int
		required                        bool
	}{
		{name: "name", min: 1, max: 1, required: true, label: "<name>", usage: "<name>"},
		{name: "userName", min: 1, max: 1, required: true, label: "<userName>", usage: "<userName>"},
		{name: "InputFile", min: 1, max: 1, required: true, label: "<InputFile>", usage: "<InputFile>"},
		{name: "log.colorMode", min: 1, max: 1, required: true, label: "<log.colorMode>", usage: "<log.colorMode>"},
		{name: "userName", min: 1, max: 1, label: "<userName>", usage: "[<userName>]"},
		{name: "inputFile", placeholder: "PathToFile", min: 1, max: 1, required: true, label: "<PathToFile>", usage: "<PathToFile>"},
		{name: "inputFile", placeholder: "<PathToFile>", min: 1, max: 1, label: "<PathToFile>", usage: "[<PathToFile>]"},
		{name: "items", min: 1, max: -1, required: true, label: "<items>...", usage: "<items>..."},
		{name: "items", min: 1, max: -1, label: "<items>...", usage: "[<items>...]"},
		{name: "coordinates", min: 2, max: 3, required: true, label: "<coordinates>{2..3}", usage: "<coordinates>{2..3}"},
		{name: "coordinates", min: 2, max: 3, label: "<coordinates>{2..3}", usage: "[<coordinates>{2..3}]"},
	} {
		t.Run(test.usage, func(t *testing.T) {
			argument := NewArgument(grammarString{min: test.min, max: test.max}, test.name).
				WithPlaceholder(test.placeholder).WithRequired(test.required)
			app := NewSimpleApplication("app").WithArgument(argument).MustCreate()
			t.Cleanup(func() { assert.NoError(t, app.Close()) })
			invocation := app.MustParse(t.Context(), nil, "--help")
			t.Cleanup(func() { assert.NoError(t, invocation.Close()) })
			data, err := buildHelpTemplateData(t.Context(), invocation)
			require.NoError(t, err)
			require.Len(t, data.Arguments, 1)
			assert.Equal(t, test.name, data.Arguments[0].Name)
			assert.Equal(t, test.label, data.Arguments[0].Label)
			assert.Equal(t, "app [flags] "+test.usage, data.Usage)
		})
	}
}

func TestMissingArgumentHelpUsesDeclaredNameAndEqualsFlagValues(t *testing.T) {
	type options struct {
		Greeting string `flag:"greeting" short:"g" default:"Hello" description:"Greeting to use."`
		Name     string `argument:"name" required:"true" description:"Name of the person to greet."`
	}
	var output bytes.Buffer
	app := NewApplication[options]("simple", "Greet someone using cli4g.").
		WithErrorHandler(NewRespondErrorHandler().MustCreate()).MustCreate()
	t.Cleanup(func() { assert.NoError(t, app.Close()) })
	ctx := ContextWithStdio(t.Context(), nil, nil, &output)
	_, err := app.Parse(ctx, nil)
	require.ErrorIs(t, err, ErrValidation)
	assert.Equal(t, `Error: required argument name is missing

simple - Greet someone using cli4g.

Usage:
  simple [flags] <name>

Arguments:
  <name>  Name of the person to greet. (required)

Flags:
  -g, --greeting=<greeting>  Greeting to use. (default: Hello)
  -h, --help                 Show help.
`, output.String())
	output.Reset()
	for _, args := range [][]string{{"--greeting=Hey", "Ada"}, {"--greeting", "Hey", "Ada"}} {
		invocation, err := app.Parse(ctx, nil, args...)
		require.NoError(t, err)
		_, actual := invocation.Application()
		assert.Equal(t, options{Greeting: "Hey", Name: "Ada"}, actual)
		assert.NoError(t, invocation.Close())
	}
	assert.Empty(t, output.String())
}
