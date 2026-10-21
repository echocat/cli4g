package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZshCompletionProtocolPreservesContextDescriptionsAndDirectives(t *testing.T) {
	var input codec.CompletionInput
	app := NewSimpleApplication("tool").WithCompletion(true).
		WithInitializeHandler(func(context.Context, NoOptions) error { panic("completion must skip initialization") }).
		WithFlag(NewFlag(codec.WithCompletionHandler(codec.NewString(), func(_ context.Context, value codec.CompletionInput) (codec.CompletionResult, error) {
			input = value
			return codec.CompletionResult{Handled: true, NoSpace: true, KeepOrder: true, Candidates: []codec.CompletionCandidate{
				{Value: "ü:zulu", Description: "First: choice\nnext line"}, {Value: "ü:alpha", Description: "Second"},
			}}, nil
		}), "value")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var output, diagnostics bytes.Buffer
	ctx := ContextWithStdio(nil, nil, &output, &diagnostics)
	require.NoError(t, app.ParseAndExecute(ctx, nil, "__complete", "1", "zsh", "1", "ü:", "--value", "ü:suffix", "--later"))
	assert.Equal(t, []string{"--value", "ü:suffix", "--later"}, input.Words)
	assert.Equal(t, "ü:", input.Prefix)
	assert.Equal(t, "suffix", input.Suffix)
	assert.Equal(t, []string{"1", "0", "1", "1", "", "", "ü:zulu", "First: choice\nnext line", "ü:alpha", "Second", ""}, strings.Split(output.String(), "\x00"))
	assert.Empty(t, diagnostics.String())
	for _, args := range [][]string{
		{"1", "zsh"}, {"1", "zsh", "-1", "", ""}, {"1", "zsh", "no-index", "", ""},
		{"1", "zsh", "0", "not-a-prefix", "value"}, {"1", "zsh", "2", "", "value"},
	} {
		output.Reset()
		require.NoError(t, app.ParseAndExecute(ctx, nil, append([]string{"__complete"}, args...)...))
		assert.Equal(t, "1\x000\x000\x000\x00\x00\x00", output.String())
		assert.Empty(t, diagnostics.String())
	}
}
