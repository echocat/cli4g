package cli

import (
	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateRejectsTaggedWholeAndExplicitDescendantBindings(t *testing.T) {
	for _, kind := range []string{"flag", "argument"} {
		t.Run(kind, func(t *testing.T) {
			leaf := NewFlag(codec.NewString()).WithName("leaf").WithField("Text").WithDefault("fallback")
			var err error
			if kind == "flag" {
				_, err = NewApplication[*struct {
					*ExplicitEmbeddedValue `flag:"whole"`
				}]().WithName("app").WithFlag(leaf).Create()
			} else {
				_, err = NewApplication[*struct {
					*ExplicitEmbeddedValue `argument:"whole"`
				}]().WithName("app").WithFlag(leaf).Create()
			}
			require.ErrorContains(t, err, "overlapping")
		})
	}
}

func TestCreateRejectsExplicitWholeDescendantBindingsInEitherOrder(t *testing.T) {
	type options struct{ *ExplicitEmbeddedValue }
	for _, kind := range []string{"flags", "arguments", "mixed"} {
		for _, wholeFirst := range []bool{true, false} {
			t.Run(kind+map[bool]string{true: "/whole-first", false: "/leaf-first"}[wholeFirst], func(t *testing.T) {
				decoder, err := codec.NewValueP(func() *ExplicitEmbeddedValue { return new(ExplicitEmbeddedValue) })
				require.NoError(t, err)
				root := NewApplication[*options]().WithName("app")
				if kind == "flags" {
					whole := NewFlag(decoder).WithName("whole").WithField("ExplicitEmbeddedValue")
					leaf := NewFlag(codec.NewString()).WithName("leaf").WithField("Text").WithDefault("fallback")
					if wholeFirst {
						root.WithFlag(whole, leaf)
					} else {
						root.WithFlag(leaf, whole)
					}
				} else if kind == "arguments" {
					whole := NewArgument(decoder).WithName("whole").WithField("ExplicitEmbeddedValue")
					leaf := NewArgument(codec.NewString()).WithName("leaf").WithField("ExplicitEmbeddedValue", "Text")
					if wholeFirst {
						root.WithArgument(whole, leaf)
					} else {
						root.WithArgument(leaf, whole)
					}
				} else {
					if wholeFirst {
						root.WithFlag(NewFlag(decoder).WithName("whole").WithField("ExplicitEmbeddedValue"))
						root.WithStringArgument("leaf", func(a *ArgumentRequirement[string, string]) { a.WithField("Text") })
					} else {
						root.WithStringFlag("leaf", func(f *FlagRequirement[string, string]) { f.WithField("Text") })
						root.WithArgument(NewArgument(decoder).WithName("whole").WithField("ExplicitEmbeddedValue"))
					}
				}
				_, err = root.Create()
				require.ErrorContains(t, err, "overlapping")
			})
		}
	}
}

func TestSiblingBindingsAndExplicitLeafTagPrecedenceRemainValid(t *testing.T) {
	type fields struct {
		First  string `flag:"first-tag" default:"tag"`
		Second string `flag:"second-tag" default:"sibling"`
	}
	type options struct{ Fields fields }
	app, err := NewApplication[*options]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).
		WithStringFlag("first", func(f *FlagRequirement[string, string]) { f.WithField("Fields", "First") }).
		WithStringFlag("second", func(f *FlagRequirement[string, string]) { f.WithField("Fields", "Second") }).Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--first=one", "--second=two")
	require.NoError(t, err)
	defer func() { require.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, "one", value.Fields.First)
	assert.Equal(t, "two", value.Fields.Second)
}
