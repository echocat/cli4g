package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpLayout(t *testing.T) {
	for _, test := range []struct {
		name, input, expected string
		width                 int
	}{
		{"empty", "", "", 80},
		{"plain text and hard breaks", "Heading\n\n  First line.\n  Second line.\n", "Heading\n\n  First line.\n  Second line.\n", 80},
		{"commands and aliases", "  audit\tInspect.\n  serve, run\tRun.\n  empty\t\n", "  audit       Inspect.\n  serve, run  Run.\n  empty\n", 80},
		{"independent sections", "  long-command\tRun.\n\n  x\tValue.\n", "  long-command  Run.\n\n  x  Value.\n", 80},
		{"nested enum choices", "  --alpha\tone\n    a\tA.\n    longer\tLonger.\n  --b\ttwo\n", "  --alpha  one\n    a       A.\n    longer  Longer.\n  --b      two\n", 80},
		{"metadata without description", "  --name NAME\t (env: NAME) (default: world)\n  --required VALUE\t (required)\n", "  --name NAME       (env: NAME) (default: world)\n  --required VALUE  (required)\n", 80},
		{"hanging paragraph indent", "  alpha beta gamma delta epsilon\n\n  second\n", "  alpha beta gamma\n  delta epsilon\n\n  second\n", 20},
		{"hanging column indent", "  x\tone two three four five\n", "  x  one two three\n     four five\n", 20},
		{"oversized label", "  extraordinarily-long-command\tFirst words stay visible.\n  go\tGo now.\n", "  extraordinarily-long-command\n    First words stay\n    visible.\n  go  Go now.\n", 24},
		{"UTF-8 columns", "  größe\tGrüße aus Köln.\n  x\tAnderer Wert.\n", "  größe  Grüße aus Köln.\n  x      Anderer Wert.\n", 80},
		{"UTF-8 wrapping", "Grüße aus Köln an alle\n", "Grüße aus Köln\nan alle\n", 14},
		{"three columns", "  a\tX\tFirst\n  long\tYZ\tSecond\n", "  a     X   First\n  long  YZ  Second\n", 80},
		{"preserve spaces within lines", "  alpha   beta gamma\n", "  alpha   beta\n  gamma\n", 15},
		{"preserve indivisible words", "  https://example.org/long-path\n", "  https://example.org/long-path\n", 10},
		{"tiny width", "  x\ta b\n", "  x\n    a\n    b\n", 1},
		{"no final newline", "a\tb\nlong\tc", "a     b\nlong  c", 80},
		{"CRLF", "A\r\n  x\ty\r\n", "A\n  x  y\n", 80},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := formatHelp(t.Context(), []byte(test.input), test.width)
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(actual))
			assert.True(t, utf8.Valid(actual))
			assert.NotContains(t, string(actual), "\t")
		})
	}
}

func TestHelpLayoutCancellationReturnsNoOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	text, err := formatHelp(ctx, []byte("  name\tdescription\n"), 80)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, text)
}

func TestDefaultHelpFormatsBifroestStyleCommandsAndFlags(t *testing.T) {
	var output bytes.Buffer
	r := NewSimpleApplication("bifroest", "SSH server and gateway for standard SSH clients: OIDC Device Authorization, Docker/Kubernetes sessions, verifiable audit and recording.")
	for _, command := range []struct{ name, description string }{
		{"audit", "Inspect and verify signed audit journals."},
		{"flow", "Inspect configured flows."},
		{"session", "Inspect Bifröst sessions."},
		{"recording", "Inspect sealed session recordings."},
		{"version", "Show Bifröst version details."},
		{"forever", "This is a supporting command which simply runs forever until it receives a interrupt signal."},
		{"key", "Generate, export and import SSH keys."},
	} {
		r.WithCommand(NewSimpleCommand(command.name, command.description))
	}
	r.WithCommand(NewSimpleCommand("serve", "Runs the service.").WithAliases("run"))
	r.WithStringFlag("log.level", func(f *FlagRequirement[string, string]) { f.WithDescription("Minimum log level.") })
	r.WithStringFlag("log.format", func(f *FlagRequirement[string, string]) { f.WithDescription("Log formatter.") })
	r.WithStringFlag("log.colorMode", func(f *FlagRequirement[string, string]) { f.WithDescription("Color mode: auto, always or never.") })
	app := r.MustCreate()
	t.Cleanup(func() { assert.NoError(t, app.Close()) })
	ctx := ContextWithStdio(t.Context(), nil, nil, &output)
	require.NoError(t, app.ParseAndExecute(ctx, nil, "--help"))
	text := output.String()
	assert.Contains(t, text, `Commands:
  audit       Inspect and verify signed audit journals.
  flow        Inspect configured flows.
  session     Inspect Bifröst sessions.
  recording   Inspect sealed session recordings.
  version     Show Bifröst version details.
  forever     This is a supporting command which simply runs forever until it
              receives a interrupt signal.
  key         Generate, export and import SSH keys.
  serve, run  Runs the service.
`)
	for _, description := range []string{"Minimum log level.", "Log formatter.", "Color mode: auto, always or never.", "Show help."} {
		for line := range strings.Lines(text) {
			if strings.Contains(line, description) {
				assert.Equal(t, 26, strings.Index(line, description), line)
			}
		}
		assert.Contains(t, text, description)
	}
	for line := range strings.Lines(text) {
		assert.LessOrEqual(t, utf8.RuneCountInString(strings.TrimSuffix(line, "\n")), 80, line)
		assert.NotContains(t, line, "\t")
		assert.Equal(t, strings.TrimRight(strings.TrimSuffix(line, "\n"), " "), strings.TrimSuffix(line, "\n"))
	}
}

func TestTemplatedHelpWidthAndFormattingSnapshots(t *testing.T) {
	var output bytes.Buffer
	r := NewTemplatedHelpHandler().WithOutput(&output).WithWidth(16).
		WithApplicationTemplate("{{.FullName}}\n  x\tone two three four\n")
	help := r.MustCreate()
	r.SetWidth(120)
	r.SetFormatting(false)
	raw := r.MustCreate()
	app := NewSimpleApplication("example").MustCreate()
	t.Cleanup(func() { assert.NoError(t, app.Close()) })
	invocation := app.MustParse(t.Context(), nil, "--help")
	t.Cleanup(func() { assert.NoError(t, invocation.Close()) })
	require.NoError(t, help(t.Context(), invocation))
	assert.Equal(t, "example\n  x  one two\n     three four\n", output.String())
	output.Reset()
	require.NoError(t, raw(t.Context(), invocation))
	assert.Equal(t, "example\n  x\tone two three four\n", output.String())
	_, err := r.WithWidth(-1).Create()
	assert.ErrorContains(t, err, "help width must not be negative")
}
