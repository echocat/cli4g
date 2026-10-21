package cli

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type completingString struct {
	codec.Codec[string, string]
	complete codec.CompletionFunc
}

func TestCompletionCancelsAnActiveProvider(t *testing.T) {
	started := make(chan struct{})
	provider := completingString{Codec: codec.NewString(), complete: func(ctx context.Context, _ codec.CompletionInput) (codec.CompletionResult, error) {
		close(started)
		<-ctx.Done()
		return codec.CompletionResult{}, ctx.Err()
	}}
	app := NewSimpleApplication("tool").WithFlag(NewFlag(provider, "value")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := app.Complete(ctx, nil, completionAtEnd("--value", ""))
		finished <- err
	}()
	<-started
	cancel()
	require.ErrorIs(t, <-finished, context.Canceled)
}

func (source completingString) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, source)
}

func (source completingString) Complete(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	return source.complete(ctx, input)
}

func completionAtEnd(words ...string) CompletionRequest {
	if len(words) == 0 {
		return CompletionRequest{}
	}
	return CompletionRequest{Words: words, Word: len(words) - 1, Offset: len(words[len(words)-1])}
}

func completionTexts(result codec.CompletionResult) []string {
	values := make([]string, len(result.Candidates))
	for i, candidate := range result.Candidates {
		values[i] = candidate.Value
	}
	return values
}

func TestCompletionNamesEnumsCollectionsAndFiles(t *testing.T) {
	choice, err := codec.NewEnum(codec.NewString(), "production", "preview", "development")
	require.NoError(t, err)
	app := NewApplication[struct {
		Namespace string            `flag:"namespace|ns" short:"n" enum:"production|preview|development"`
		Verbose   bool              `flag:"verbose" short:"v"`
		Hidden    bool              `flag:"secret" visible:"false"`
		Labels    []string          `flag:"label"`
		Choices   []string          `flag:"choice" type:"choices"`
		Mapping   map[string]string `flag:"mapping" type:"mapping"`
		Input     *os.File          `flag:"input"`
	}]("tool").WithErrorHandlerEnabled(false).
		WithCodec(codec.NewSlice(choice), "choices").
		WithCodec(codec.NewMap[string](choice), "mapping").
		WithCommand(NewSimpleCommand("deploy").WithAliases("push"), NewSimpleCommand("internal").WithVisible(false)).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		words []string
		want  []string
		files bool
	}{
		{[]string{"dep"}, []string{"deploy"}, false},
		{[]string{"pu"}, []string{"push"}, false},
		{[]string{"--na"}, []string{"--namespace"}, false},
		{[]string{"--sec"}, []string{}, false},
		{[]string{"--namespace", "pr"}, []string{"production", "preview"}, false},
		{[]string{"--ns=pr"}, []string{"--ns=production", "--ns=preview"}, false},
		{[]string{"-npr"}, []string{"-nproduction", "-npreview"}, false},
		{[]string{"-vnpr"}, []string{"-vnproduction", "-vnpreview"}, false},
		{[]string{"--verbose=f"}, []string{"--verbose=false"}, false},
		{[]string{"--namespace", "production", "--na"}, []string{}, false},
		{[]string{"--label", "a", "--lab"}, []string{"--label"}, false},
		{[]string{"--choice", "pr"}, []string{"production", "preview"}, false},
		{[]string{"--mapping", "env=pr"}, []string{"env=production", "env=preview"}, false},
		{[]string{"--mapping=env=pr"}, []string{"--mapping=env=production", "--mapping=env=preview"}, false},
		{[]string{"--input", "missing"}, []string{}, true},
		{[]string{"--", "--na"}, []string{}, true},
	} {
		t.Run(test.words[len(test.words)-1], func(t *testing.T) {
			result, err := app.Complete(nil, nil, completionAtEnd(test.words...))
			require.NoError(t, err)
			assert.Equal(t, test.want, completionTexts(result))
			assert.Equal(t, test.files, result.Files)
		})
	}
}

func TestCompletionMultiTokenEnumsMatchEarlierValues(t *testing.T) {
	locations, err := codec.NewEnum(codec.NewStrings(),
		[]string{"eu", "production"}, []string{"eu", "preview"}, []string{"us", "sandbox"})
	require.NoError(t, err)
	app := NewSimpleApplication("tool").WithArgument(NewArgument(locations, "location")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	result, err := app.Complete(nil, nil, completionAtEnd("eu", "pr"))
	require.NoError(t, err)
	assert.Equal(t, []string{"production", "preview"}, completionTexts(result))
	result, err = app.Complete(nil, nil, completionAtEnd("us", ""))
	require.NoError(t, err)
	assert.Equal(t, []string{"sandbox"}, completionTexts(result))
}

func TestCompletionCodecSeesDependenciesScopesAndInputAfterCursor(t *testing.T) {
	var captured codec.CompletionInput
	var contextValue codec.CompletionValue
	provider := completingString{Codec: codec.NewString(), complete: func(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
		captured = input
		var err error
		contextValue, _, err = input.Effective.Flag(ctx, "ctx")
		return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "production"}}}, err
	}}
	app := NewApplication[struct {
		Context string `flag:"context|ctx" short:"c" env:"CLUSTER_CONTEXT" default:"default-cluster"`
	}]("tool").WithErrorHandlerEnabled(false).
		WithCreateOptionsHandler(func(context.Context) (struct {
			Context string `flag:"context|ctx" short:"c" env:"CLUSTER_CONTEXT" default:"default-cluster"`
		}, error) {
			panic("completion must not construct options")
		}).
		WithCommand(NewCommand[struct {
			Namespace string   `flag:"namespace" short:"n" type:"namespace"`
			Labels    []string `flag:"label"`
			Resource  string   `argument:"resource" required:"true"`
		}]("get").WithCodec(provider, "namespace")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	words := []string{"get", "pods", "--namespace", "pro", "-c", "cluster-b", "--label", "", "--label", "web", "--unknown"}
	result, err := app.Complete(nil, nil, CompletionRequest{Words: words, Word: 3, Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, []string{"production"}, completionTexts(result))
	assert.Equal(t, []string{"tool", "get"}, captured.CommandPath)
	assert.Equal(t, codec.CompletionTarget{Kind: codec.CompletionFlagValue, Name: "namespace", Scope: 1}, captured.Target)
	assert.Equal(t, []string{"cluster-b"}, captured.Flags["context"][0].Values)
	assert.Equal(t, []string{""}, captured.Flags["label"][0].Values)
	assert.Equal(t, []string{"web"}, captured.Flags["label"][1].Values)
	assert.Equal(t, []string{"pods"}, captured.Arguments["resource"])
	assert.Equal(t, []string{"pods"}, captured.Positionals)
	assert.Equal(t, words, captured.Words)
	assert.Equal(t, codec.CompletionCLI, contextValue.Source)
	assert.Equal(t, []string{"cluster-b"}, contextValue.Values)
	assert.Contains(t, captured.Scopes[0].Flags, "context")
	assert.NotContains(t, captured.Scopes[1].Flags, "context")
	_, err = app.Complete(nil, nil, completionAtEnd("get", "pods", "--context", "--namespace", "pro"))
	require.NoError(t, err)
	assert.True(t, captured.Flags["context"][0].Missing)
	assert.Empty(t, contextValue.Values)
}

func BenchmarkCompletion(b *testing.B) {
	app := NewApplication[struct {
		Namespace string `flag:"namespace" enum:"production|preview|development"`
	}]("tool").MustCreate()
	b.Cleanup(func() { require.NoError(b, app.Close()) })
	request := completionAtEnd("--namespace", "pr")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := app.Complete(context.Background(), nil, request); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCompletionEffectivePrecedenceAndLazyDefaults(t *testing.T) {
	var value codec.CompletionValue
	lookups := 0
	provider := completingString{Codec: codec.NewString(), complete: func(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
		var err error
		value, _, err = input.Effective.Flag(ctx, "context")
		return codec.CompletionResult{Handled: true}, err
	}}
	app := NewApplication[struct {
		Context   string `flag:"context" env:"CONTEXT" default:"default-cluster"`
		Namespace string `flag:"namespace" type:"namespace"`
		Invalid   int    `flag:"invalid" default:"not-an-int"`
	}]("tool").WithCodec(provider, "namespace").MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { lookups++; return "env-cluster", true }}
	_, err := app.Complete(nil, opts, completionAtEnd("--namespace", ""))
	require.NoError(t, err)
	assert.Equal(t, codec.CompletionEnvironment, value.Source)
	assert.Equal(t, []string{"env-cluster"}, value.Values)
	assert.Equal(t, "CONTEXT", value.Environment)
	assert.Equal(t, 1, lookups)
	_, err = app.Complete(nil, opts, completionAtEnd("--context=", "--namespace", ""))
	require.NoError(t, err)
	assert.Equal(t, codec.CompletionCLI, value.Source)
	assert.Equal(t, []string{""}, value.Values)
	assert.Equal(t, 1, lookups)
	opts.LookupEnv = func(context.Context, string) (string, bool) { return "", false }
	_, err = app.Complete(nil, opts, completionAtEnd("--namespace", ""))
	require.NoError(t, err)
	assert.Equal(t, codec.CompletionDefault, value.Source)
	assert.Equal(t, []string{"default-cluster"}, value.Values)
}

func TestCompletionProfilesErrorsCancellationAndReuse(t *testing.T) {
	failure := errors.New("provider unavailable")
	provider := completingString{Codec: codec.NewString(), complete: func(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
		if input.Prefix == "fail" {
			return codec.CompletionResult{}, failure
		}
		if input.Prefix == "cancel" {
			<-ctx.Done()
			return codec.CompletionResult{}, ctx.Err()
		}
		return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: input.Prefix + "-ok"}}}, nil
	}}
	app := NewSimpleApplication("tool").WithCodec(provider, "custom").
		WithFlag(NewFlag(provider, "value")).
		WithFlag(NewFlag(codec.NewBool(), "profiled").WithProfiles("test")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Complete(nil, nil, CompletionRequest{Word: -1})
	require.Error(t, err)
	_, err = app.Complete(nil, &ParseOpts{Profiles: []string{"missing"}}, CompletionRequest{})
	require.Error(t, err)
	result, err := app.Complete(nil, nil, completionAtEnd("--prof"))
	require.NoError(t, err)
	assert.Empty(t, result.Candidates)
	result, err = app.Complete(nil, &ParseOpts{Profiles: []string{"test"}}, completionAtEnd("--prof"))
	require.NoError(t, err)
	assert.Equal(t, []string{"--profiled"}, completionTexts(result))
	_, err = app.Complete(nil, nil, completionAtEnd("--value", "fail"))
	require.ErrorIs(t, err, failure)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = app.Complete(ctx, nil, completionAtEnd("--value", "cancel"))
	require.ErrorIs(t, err, context.Canceled)
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			result, err := app.Complete(nil, nil, completionAtEnd("--value", "parallel"))
			assert.NoError(t, err)
			assert.Equal(t, []string{"parallel-ok"}, completionTexts(result))
		})
	}
	workers.Wait()
	require.NoError(t, app.Close())
	_, err = app.Complete(nil, nil, CompletionRequest{})
	require.ErrorIs(t, err, ErrApplicationClosed)
}
