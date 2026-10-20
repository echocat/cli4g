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

func TestFishCompletionInputPreservesLiteralWordsAndCursor(t *testing.T) {
	for _, test := range []struct {
		line string
		want CompletionRequest
	}{
		{"tool ", completionAtEnd("")},
		{"tool --format=ya", completionAtEnd("--format=ya")},
		{`tool --format 'two wo`, completionAtEnd("--format", "two wo")},
		{`tool --format="two wo`, completionAtEnd("--format=two wo")},
		{`tool --context '' --namespace `, completionAtEnd("--context", "", "--namespace", "")},
		{`tool --context 'a\'b' --namespace `, completionAtEnd("--context", "a'b", "--namespace", "")},
		{`tool --context "a\"b" --namespace `, completionAtEnd("--context", `a"b`, "--namespace", "")},
		{`tool --context a\ b --namespace `, completionAtEnd("--context", "a b", "--namespace", "")},
		{`tool --context a\nb --namespace `, completionAtEnd("--context", "a\nb", "--namespace", "")},
		{`tool --context '\n' --namespace `, completionAtEnd("--context", `\n`, "--namespace", "")},
		{`tool --context "\n" --namespace `, completionAtEnd("--context", `\n`, "--namespace", "")},
		{`tool --context \u00fcber --namespace `, completionAtEnd("--context", "über", "--namespace", "")},
		{`tool --context \U0001f600 --namespace `, completionAtEnd("--context", "😀", "--namespace", "")},
		{`tool --context \x61\141\cb --namespace `, completionAtEnd("--context", "aa\x02", "--namespace", "")},
		{"tool --context a\\\nb --namespace ", completionAtEnd("--context", "ab", "--namespace", "")},
		{"tool --context 'a\nb' --namespace ", completionAtEnd("--context", "a\nb", "--namespace", "")},
		{`tool --context (echo "a b") --namespace `, completionAtEnd("--context", `(echo "a b")`, "--namespace", "")},
		{`tool --context (echo (echo a)) --namespace `, completionAtEnd("--context", `(echo (echo a))`, "--namespace", "")},
		{`tool --context $cluster --namespace `, completionAtEnd("--context", "$cluster", "--namespace", "")},
		{`command tool --context cluster-b --namespace cl`, completionAtEnd("--context", "cluster-b", "--namespace", "cl")},
		{`CONTEXT=cluster-b exec tool --namespace cl`, completionAtEnd("--namespace", "cl")},
		{"ignored | tool --format ü", completionAtEnd("--format", "ü")},
	} {
		t.Run(test.line, func(t *testing.T) {
			request, err := fishCompletionRequest(test.line, len(test.line))
			require.NoError(t, err)
			assert.Equal(t, test.want, request)
		})
	}
	line := "tool --namespace üsuffix --context über"
	request, err := fishCompletionRequest(line, len("tool --namespace ü"))
	require.NoError(t, err)
	assert.Equal(t, CompletionRequest{Words: []string{"--namespace", "üsuffix", "--context", "über"}, Word: 1, Offset: 2}, request)
	_, err = fishCompletionRequest(line, -1)
	require.Error(t, err)
	_, err = fishCompletionRequest(line, len(line)+1)
	require.Error(t, err)
	_, err = fishCompletionRequest("tool", 4)
	require.Error(t, err)
}

func TestFishCompletionProtocolPreservesNewlinesAndSuppressesInvalidRequests(t *testing.T) {
	var input codec.CompletionInput
	app := NewSimpleApplication("tool").WithCompletion(true).
		WithInitializeHandler(func(context.Context, NoOptions) error { panic("completion must skip initialization") }).
		WithFlag(NewFlag(codec.WithCompletionHandler(codec.NewString(), func(_ context.Context, value codec.CompletionInput) (codec.CompletionResult, error) {
			input = value
			return codec.CompletionResult{Handled: true, NoSpace: true, KeepOrder: true, Candidates: []codec.CompletionCandidate{
				{Value: "a\nb-value", Description: "First\nsecond"}, {Value: "a\nb\tunsupported"},
			}}, nil
		}), "value")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var output, diagnostics bytes.Buffer
	ctx := ContextWithStdio(nil, nil, &output, &diagnostics)
	line := "tool --value 'a\nb' --later"
	prefix := "tool --value 'a\nb"
	require.NoError(t, app.ParseAndExecute(ctx, nil, "__complete", "1", "fish", prefix+"\n", line+"\n"))
	assert.Equal(t, []string{"--value", "a\nb", "--later"}, input.Words)
	assert.Equal(t, "a\nb", input.Prefix)
	assert.Equal(t, []string{"1", "0", "1", "1", "", "", "a\nb-value", "First\nsecond", ""}, strings.Split(output.String(), "\x00"))
	assert.Empty(t, diagnostics.String())
	output.Reset()
	// A real newline immediately before the cursor survives the separate printing
	// newline appended by Fish's commandline builtin.
	require.NoError(t, app.ParseAndExecute(ctx, nil, "__complete", "1", "fish", "tool --value 'a\n\n", line+"\n"))
	assert.Equal(t, "a\n", input.Prefix)
	assert.Equal(t, "b", input.Suffix)
	assert.Contains(t, output.String(), "a\nb-value\x00")
	for _, args := range [][]string{{"1", "fish"}, {"1", "fish", "different\n", "tool --value a\n"}, {"1", "fish", "tool\n", "tool\n"}} {
		output.Reset()
		require.NoError(t, app.ParseAndExecute(ctx, nil, append([]string{"__complete"}, args...)...))
		assert.Equal(t, "1\x000\x000\x000\x00\x00\x00", output.String())
		assert.Empty(t, diagnostics.String())
	}
}
