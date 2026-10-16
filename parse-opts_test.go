package cli

import (
	"context"
	"os"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupEnvReceivesCallerContextAndYieldsToExplicitFlags(t *testing.T) {
	type contextKey struct{}
	type options struct {
		Value string `flag:"value" env:"VALUE" required:"true"`
	}
	app, err := NewApplication[options]("app").
		Create()
	require.NoError(t, err)
	ctx := context.WithValue(context.Background(), contextKey{}, "parse")
	lookups := 0
	opts := &ParseOpts{LookupEnv: func(received context.Context, name string) (string, bool) {
		assert.Equal(t, ctx.Done(), received.Done())
		assert.Same(t, os.Stdin, codec.StdinOfContext(received))
		assert.Same(t, os.Stdout, codec.StdoutOfContext(received))
		assert.Equal(t, "parse", received.Value(contextKey{}))
		assert.Equal(t, "VALUE", name)
		lookups++
		return "environment", true
	}}
	invocation, err := app.Parse(ctx, opts)
	require.NoError(t, err)
	_, value := invocation.Application()
	assert.Equal(t, "environment", value.Value)
	require.NoError(t, invocation.Close())
	invocation, err = app.Parse(ctx, opts, "--value=explicit")
	require.NoError(t, err)
	_, value = invocation.Application()
	assert.Equal(t, "explicit", value.Value)
	assert.Equal(t, 1, lookups)
	require.NoError(t, invocation.Close())
}

func TestEnvironmentLookupDefaultsAndOverrides(t *testing.T) {
	const envName = "BIFROEST_CLI_TEST_ENV_VALUE"
	t.Setenv(envName, "os-value")
	type options struct {
		Value string `flag:"value" env:"BIFROEST_CLI_TEST_ENV_VALUE" required:"true"`
	}
	app, err := NewApplication[options]("app").
		Create()
	require.NoError(t, err)
	for _, test := range []struct {
		name  string
		opts  *ParseOpts
		args  []string
		value string
	}{
		{name: "nil options", value: "os-value"},
		{name: "nil callback", opts: &ParseOpts{}, value: "os-value"},
		{
			name: "custom lookup replaces OS lookup",
			opts: &ParseOpts{LookupEnv: func(ctx context.Context, name string) (string, bool) {
				assert.NotNil(t, ctx)
				assert.Equal(t, envName, name)
				return "custom-value", true
			}},
			value: "custom-value",
		},
		{
			name: "explicit CLI value overrides OS value",
			args: []string{"--value=cli-value"}, value: "cli-value",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(nil, test.opts, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, value := invocation.Application()
			assert.Equal(t, test.value, value.Value)
		})
	}
	t.Run("custom lookup does not fall back to OS", func(t *testing.T) {
		opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
			return "", false
		}}
		invocation, err := app.Parse(context.Background(), opts)
		require.ErrorContains(t, err, "required flag --value is missing")
		assert.Nil(t, invocation)
	})
	t.Run("empty but present OS value", func(t *testing.T) {
		t.Setenv(envName, "")
		invocation, err := app.Parse(context.Background(), nil)
		require.NoError(t, err)
		defer func() { assert.NoError(t, invocation.Close()) }()
		_, value := invocation.Application()
		assert.Empty(t, value.Value)
	})
}
