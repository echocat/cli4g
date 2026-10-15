package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFields(t *testing.T) {
	type options struct{ testFlagsOptsMixin }
	a, err := NewApplication[*options]().
		WithName("embedded").
		WithFlag(
			NewFlag(custom1Codec{}).
				WithName("custom1").
				WithField("testFlagsOptsMixin", "Custom1"),
			NewFlag(customVariadic1Codec{}).
				WithName("customVariadic1").
				WithField("testFlagsOptsMixin", "CustomVariadic1"),
		).
		Create()
	require.NoError(t, err)
	i, err := a.Parse(context.Background(), nil, "--requiredString", "yes", "--someString", "hello")
	require.NoError(t, err)
	_, o := i.Application()
	assert.Equal(t, "hello", o.SomeString)
}

func TestExplicitCodecOverridesTaggedField(t *testing.T) {
	type opts struct {
		Custom custom1 `flag:"custom"`
	}
	a, err := NewApplication[*opts]().
		WithName("custom").
		WithFlag(
			NewFlag(custom1Codec{}).
				WithName("custom").
				WithField("Custom"),
		).
		Create()
	require.NoError(t, err)
	i, err := a.Parse(context.Background(), nil, "--custom", "works")
	require.NoError(t, err)
	_, o := i.Application()
	assert.Equal(t, "works", o.Custom.v)
}
