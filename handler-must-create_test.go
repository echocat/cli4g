package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerMustCreateBuildsUsableReturningHandlers(t *testing.T) {
	var output bytes.Buffer
	response := NewRespondErrorHandler().WithOutput(&output).MustCreate()
	failure := errors.New("failed")
	assert.Same(t, failure, response(context.Background(), nil, failure))
	assert.Equal(t, "Error: failed\n", output.String())
	output.Reset()
	help := NewTemplatedHelpHandler().WithOutput(&output).
		WithApplicationTemplate("HELP {{.FullName}}\n").MustCreate()
	app := NewSimpleApplication().WithName("app").WithHelpHandler(help).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(context.Background()))
	assert.Equal(t, "HELP app\n", output.String())
}

func TestHandlerMustCreatePanicsOnInvalidConfiguration(t *testing.T) {
	var response *RespondErrorHandlerRequirement
	var help *TemplatedHelpHandlerRequirement
	assert.Panics(t, func() { response.MustCreate() })
	assert.Panics(t, func() { help.MustCreate() })
	assert.Panics(t, func() { NewTemplatedHelpHandler().WithApplicationTemplate("{{").MustCreate() })
}
