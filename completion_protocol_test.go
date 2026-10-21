package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionProtocolSkipsLifecycleAndUsesContextStreams(t *testing.T) {
	app := NewSimpleApplication("tool").WithCompletion(true).WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(codec.NewBool(), "verbose")).
		WithHandler(func(context.Context, NoOptions) error { panic("not a normal invocation") }).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var output bytes.Buffer
	ctx := ContextWithStdio(nil, strings.NewReader(`{"words":["--ver"],"word":0,"offset":5}`), &output, nil)
	invocation, err := app.Parse(ctx, nil, DefaultCompletionRequestCommandName, "1", "nu")
	require.NoError(t, err)
	require.Empty(t, output.String())
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(ctx))
	var result codec.CompletionResult
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, []string{"--verbose"}, completionTexts(result))
	output.Reset()
	require.NoError(t, app.ParseAndExecute(ctx, nil, DefaultCompletionRequestCommandName, "1", "bash", "10", "tool --ver", "--ver"))
	assert.Equal(t, []string{"1", "0", "0", "0", "", "", "", "--verbose", ""}, strings.Split(output.String(), "\x00"))
}

func TestCompletionProtocolRejectsMalformedRequestsQuietly(t *testing.T) {
	app := NewSimpleApplication("tool").WithCompletion(true).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, input := range []string{``, `{`, `{"words":null,"word":-1}`, `{} {}`, `{"unexpected":true}`} {
		var output, diagnostics bytes.Buffer
		ctx := ContextWithStdio(nil, strings.NewReader(input), &output, &diagnostics)
		require.NoError(t, app.ParseAndExecute(ctx, nil, DefaultCompletionRequestCommandName, "1", "nu"))
		var result codec.CompletionResult
		require.NoError(t, json.Unmarshal(output.Bytes(), &result))
		assert.True(t, result.Handled)
		assert.Empty(t, result.Candidates)
		assert.Empty(t, diagnostics.String())
	}
}

func TestCompletionInstallationIsIdempotentAndProtectsExistingFiles(t *testing.T) {
	app := NewSimpleApplication("tool").WithCompletionRequestCommandName(DefaultCompletionRequestCommandName).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, shell := range []CompletionShell{CompletionBash, CompletionFish, CompletionNushell, CompletionPowerShell, CompletionZsh} {
		directory := t.TempDir()
		path, err := app.InstallCompletion(nil, shell, directory)
		require.NoError(t, err)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(contents), completionScriptHeader)
		again, err := app.InstallCompletion(nil, shell, directory)
		require.NoError(t, err)
		assert.Equal(t, path, again)
		require.NoError(t, os.WriteFile(path, []byte("user-managed"), 0600))
		_, err = app.InstallCompletion(nil, shell, directory)
		require.ErrorContains(t, err, "not managed by cli4g")
		contents, err = os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "user-managed", string(contents))
	}
	t.Setenv("BASH_COMPLETION_USER_DIR", t.TempDir())
	directory, err := completionDirectory(context.Background(), CompletionBash)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(os.Getenv("BASH_COMPLETION_USER_DIR"), "completions"), directory)
	_, err = app.CompletionScript("unsupported-shell")
	require.Error(t, err)
	_, err = NewSimpleApplication("../bad").MustCreate().InstallCompletion(nil, CompletionBash, t.TempDir())
	require.Error(t, err)
}

func TestPowerShellCompletionProtocolPreservesUnicodeAndRejectsMalformedInput(t *testing.T) {
	app := NewApplication[struct {
		Format string `flag:"format" enum:"über|two words"`
	}]("tool").WithCompletion(true).WithInitializeHandler(func(context.Context, struct {
		Format string `flag:"format" enum:"über|two words"`
	}) error {
		panic("completion must skip initialization")
	}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		input string
		want  []string
	}{
		{base64.StdEncoding.EncodeToString([]byte(`{"words":["--format","ü"],"word":1,"offset":2}`)), []string{"über"}},
		{base64.StdEncoding.EncodeToString([]byte(`{"words":["--format","two wo"],"word":1,"offset":6}`)), []string{"two words"}},
		{"not base64!", []string{}},
		{base64.StdEncoding.EncodeToString([]byte(`{} {}`)), []string{}},
		{base64.StdEncoding.EncodeToString([]byte(`{"unexpected":true}`)), []string{}},
		{base64.StdEncoding.EncodeToString([]byte(`{"words":["--format","ü"],"word":1,"offset":2}` + strings.Repeat(" ", 1<<20))), []string{}},
	} {
		var output, diagnostics bytes.Buffer
		ctx := ContextWithStdio(nil, strings.NewReader(test.input), &output, &diagnostics)
		require.NoError(t, app.ParseAndExecute(ctx, nil, DefaultCompletionRequestCommandName, "1", "powershell"))
		decoded, err := base64.StdEncoding.DecodeString(output.String())
		require.NoError(t, err)
		var result codec.CompletionResult
		require.NoError(t, json.Unmarshal(decoded, &result))
		assert.Equal(t, test.want, completionTexts(result))
		assert.Empty(t, diagnostics.String())
	}
}

func TestCompletionDoesNotReserveTheFormerProtocolFlag(t *testing.T) {
	app := NewSimpleApplication("tool").WithCompletion(true).
		WithHandler(func(context.Context, NoOptions) error { return nil }).
		WithErrorHandlerEnabled(false).WithStringFlag("cli4g-completion").MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--cli4g-completion", "ordinary-value")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("cli4g-completion")
	assert.Equal(t, "ordinary-value", value)
}

func TestBashCompletionInputQuotesSeparatorsUnicodeAndCursor(t *testing.T) {
	for _, test := range []struct {
		line  string
		point int
		want  CompletionRequest
	}{
		{"tool --format=ya", len("tool --format=ya"), completionAtEnd("--format=ya")},
		{"tool --format 'two wo", len("tool --format 'two wo"), completionAtEnd("--format", "two wo")},
		{"tool --format two\\ wo", len("tool --format two\\ wo"), completionAtEnd("--format", "two wo")},
		{"tool --format über", len("tool --format über"), completionAtEnd("--format", "über")},
		{"other | tool --format ya", len("other | tool --format ya"), completionAtEnd("--format", "ya")},
		{"tool --format ya | other", len("tool --format ya"), completionAtEnd("--format", "ya")},
		{"tool --format yaml --context production", len("tool --format ya"), CompletionRequest{Words: []string{"--format", "yaml", "--context", "production"}, Word: 1, Offset: 2}},
		{"tool ", len("tool "), completionAtEnd("")},
		{"tool  --flag", len("tool "), CompletionRequest{Words: []string{"", "--flag"}, Word: 0}},
	} {
		t.Run(test.line, func(t *testing.T) {
			request, err := bashCompletionRequest(test.line, test.point)
			require.NoError(t, err)
			assert.Equal(t, test.want, request)
		})
	}
}

func TestBashCompletionReplacementPrefixRetainsLiteralWordBreaks(t *testing.T) {
	for _, test := range []struct{ prefix, current, want string }{
		{"--format=ya", "ya", "--format="},
		{"--format=", "", "--format="},
		{"--format=", "''", "--format="},
		{"--format=", `""`, "--format="},
		{"--format=two wo", `'two wo'`, "--format="},
		{"--format=a'b", `"a'b"`, "--format="},
		{"--format=a'b", "a'b", "--format="},
		{"--format=$(printf literal)", `'$(printf literal)'`, "--format="},
		{"colon:fi", "fi", "colon:"},
		{"--format=ü", "ü", "--format="},
		{"--format=two wo", "wrong", ""},
	} {
		t.Run(test.current, func(t *testing.T) {
			assert.Equal(t, test.want, bashCompletionReplacementPrefix(test.prefix, test.current))
		})
	}
}
