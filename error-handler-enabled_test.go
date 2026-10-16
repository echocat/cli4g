package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlerCanBeDisabledLocallyAndIsFrozenAtCreate(t *testing.T) {
	previous := DefaultErrorHandler
	t.Cleanup(func() { DefaultErrorHandler = previous })
	var output bytes.Buffer
	globalCalls, customCalls, replacementCalls := 0, 0, 0
	globalError := errors.New("global response")
	DefaultErrorHandler = func(_ context.Context, _ InvocationLike, err error) error {
		globalCalls++
		_, _ = fmt.Fprintln(&output, err)
		return errors.Join(err, globalError)
	}
	root := NewSimpleApplication("quiet").
		WithHelp(false).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { customCalls++; return err }).
		WithErrorHandlerEnabled(false)
	quiet, err := root.Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, quiet.Close()) }()
	root.SetErrorHandlerEnabled(true)
	enabled, err := root.Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, enabled.Close()) }()
	fallback, err := NewSimpleApplication("fallback").
		WithHelp(false).
		WithErrorHandler(nil).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, fallback.Close()) }()
	DefaultErrorHandler = func(_ context.Context, _ InvocationLike, err error) error { replacementCalls++; return err }
	_, err = quiet.Parse(nil, nil, "--unknown")
	require.EqualError(t, err, "unknown flag --unknown")
	assert.NotErrorIs(t, err, globalError)
	assert.Zero(t, globalCalls)
	assert.Zero(t, customCalls)
	assert.Zero(t, output.Len(), "disabled application must not emit automatic errors/help")
	_, err = enabled.Parse(nil, nil, "--unknown")
	require.EqualError(t, err, "unknown flag --unknown")
	assert.Equal(t, 1, customCalls)
	_, err = fallback.Parse(nil, nil, "--unknown")
	require.ErrorIs(t, err, globalError)
	assert.Equal(t, 1, globalCalls)
	assert.Contains(t, output.String(), "unknown flag --unknown")
	assert.Zero(t, replacementCalls, "existing applications retain their captured default")
	other, err := NewSimpleApplication("other").
		WithHelp(false).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, other.Close()) }()
	_, err = other.Parse(nil, nil, "--unknown")
	require.Error(t, err)
	assert.Equal(t, 1, replacementCalls, "other applications still use the current global fallback")
}
