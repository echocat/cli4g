package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestCreateRejectsRequiredParentArguments(t *testing.T) {
	type options struct {
		Optional string `argument:"optional"`
		Value    string `argument:"value" required:"true"`
	}
	for _, test := range []struct {
		name   string
		path   string
		create func() (Container, error)
	}{
		{
			name: "explicit application argument",
			path: "app",
			create: func() (Container, error) {
				return NewSimpleApplication("app").
					WithArgument(
						NewArgument(codec.NewString(), "optional"),
						NewArgument(codec.NewString(), "value").
							WithRequired(true),
					).
					WithCommand(NewSimpleCommand("child")).
					Create()
			},
		},
		{
			name: "tagged application argument",
			path: "app",
			create: func() (Container, error) {
				return NewApplication[*options]("app").
					WithCommand(NewSimpleCommand("child")).
					Create()
			},
		},
		{
			name: "explicit nested command argument",
			path: "app group parent",
			create: func() (Container, error) {
				return NewSimpleApplication("app").
					WithCommand(NewSimpleCommand("group").
						WithCommand(NewSimpleCommand("parent").
							WithArgument(NewArgument(codec.NewString(), "value").
								WithRequired(true),
							).
							WithCommand(NewSimpleCommand("leaf")),
						),
					).
					Create()
			},
		},
		{
			name: "tagged nested command argument",
			path: "app group parent",
			create: func() (Container, error) {
				return NewSimpleApplication("app").
					WithCommand(NewSimpleCommand("group").
						WithCommand(NewCommand[options]("parent").
							WithCommand(NewSimpleCommand("leaf")),
						),
					).
					Create()
			},
		},
		{
			name: "required variadic argument with zero minimum arity",
			path: "app",
			create: func() (Container, error) {
				return NewSimpleApplication("app").
					WithArgument(NewArgument(codec.NewStrings(), "value").
						WithRequired(true),
					).
					WithCommand(NewSimpleCommand("child")).
					Create()
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, err := test.create()
			require.EqualError(t, err, test.path+`: required positional argument "value" cannot be combined with subcommands`)
			assert.Nil(t, app)
		})
	}
}

func TestCreateAllowsOptionalParentArgumentsAndRequiredFlags(t *testing.T) {
	type rootOptions struct {
		Token string `flag:"token" required:"true"`
		Value string `argument:"root-value"`
	}
	type parentOptions struct {
		Values []string `argument:"parent-values"`
	}
	type leafOptions struct {
		Value string `argument:"leaf-value" required:"true"`
	}
	app := NewApplication[rootOptions]("app").
		WithCommand(NewCommand[*parentOptions]("parent").
			WithCommand(NewCommand[leafOptions]("leaf")),
		).
		MustCreate()
	parent, found := FindCommand[*parentOptions](app, "parent")
	require.True(t, found)
	leaf, found := FindCommand[leafOptions](app, "parent", "leaf")
	require.True(t, found)
	for _, test := range []struct {
		name       string
		args       []string
		names      []string
		rootValue  string
		parentData []string
		leafValue  string
	}{
		{
			name:      "application's own argument",
			args:      []string{"--token=token-value", "root-data"},
			names:     []string{},
			rootValue: "root-data",
		},
		{
			name:       "parent's own variadic argument",
			args:       []string{"--token=token-value", "parent", "one", "two"},
			names:      []string{"parent"},
			parentData: []string{"one", "two"},
		},
		{
			name:      "leaf's required argument with omitted parent arguments",
			args:      []string{"parent", "leaf", "--token=token-value", "leaf-data"},
			names:     []string{"parent", "leaf"},
			leafValue: "leaf-data",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(context.Background(), nil, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			assert.Equal(t, test.names, invocation.NameStack())
			_, rootData := invocation.Application()
			assert.Equal(t, "token-value", rootData.Token)
			assert.Equal(t, test.rootValue, rootData.Value)
			if len(test.names) > 0 {
				parentData, found := parent.OptionsFrom(invocation)
				require.True(t, found)
				assert.Equal(t, test.parentData, parentData.Values)
			}
			if len(test.names) > 1 {
				leafData, found := leaf.OptionsFrom(invocation)
				require.True(t, found)
				assert.Equal(t, test.leafValue, leafData.Value)
			}
		})
	}
}

func TestCreateAllowsRequiredApplicationArgumentsWithoutSubcommands(t *testing.T) {
	app, err := NewSimpleApplication("app").
		WithArgument(NewArgument(codec.NewString(), "value").
			WithRequired(true),
		).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "data")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	argument, value := invocation.Argument(0)
	require.NotNil(t, argument)
	assert.Equal(t, "value", argument.Name())
	assert.Equal(t, "data", value)
}
