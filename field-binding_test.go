package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type bindingNested struct {
	Name  string
	Value string
}

type bindingLeaf struct{ Value string }

func (v *bindingLeaf) UnmarshalText(text []byte) error { v.Value = string(text); return nil }
func (v *bindingLeaf) MarshalText() ([]byte, error)    { return []byte(v.Value), nil }

type bindingOptions struct {
	Nested *bindingNested
	Leaf   *bindingLeaf `flag:"leaf"`
}

func TestCompiledBindingsForNestedAndPointerFields(t *testing.T) {
	a, err := NewApplication[*bindingOptions]("nested").
		WithCodec(codec.NewMarshallerP(func() *bindingLeaf { return &bindingLeaf{} })).
		WithFlag(
			NewFlag(codec.NewString(), "name").
				WithField("Nested", "Name"),
		).
		WithArgument(
			NewArgument(codec.NewString(), "value").
				WithField("Nested", "Value"),
		).
		Create()
	require.NoError(t, err)
	ctx := context.Background()
	empty, err := a.Parse(ctx, nil)
	require.NoError(t, err)
	_, emptyOptions := empty.Application()
	assert.Nil(t, emptyOptions.Nested)
	assert.Nil(t, emptyOptions.Leaf)

	first, err := a.Parse(ctx, nil, "--name=first", "--leaf=alpha", "argument")
	require.NoError(t, err)
	_, firstOptions := first.Application()
	got := firstOptions
	require.NotNil(t, got.Nested)
	require.NotNil(t, got.Leaf)
	assert.Equal(t, "first", got.Nested.Name)
	assert.Equal(t, "argument", got.Nested.Value)
	assert.Equal(t, "alpha", got.Leaf.Value)

	second, err := a.Parse(ctx, nil, "--name=second")
	require.NoError(t, err)
	_, secondOptions := second.Application()
	other := secondOptions
	require.NotNil(t, other.Nested)
	assert.Equal(t, "second", other.Nested.Name)
	assert.Nil(t, other.Leaf)
	assert.NotSame(t, got.Nested, other.Nested)
}

func TestInvalidExplicitBindingFailsAtCreate(t *testing.T) {
	type options struct{ hidden string } //nolint:unused // Deliberately unexported field inspected through reflection.
	_, err := NewApplication[*options]("invalid").
		WithFlag(
			NewFlag(codec.NewString(), "hidden").
				WithField("hidden"),
		).
		Create()
	require.ErrorContains(t, err, "must be exported")
}
