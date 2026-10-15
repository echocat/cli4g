package cli

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpAndErrorHandlersShareNonConcurrentOutput(t *testing.T) {
	var output bytes.Buffer
	var apps []*Application[NoOptions]
	for range 2 {
		help, err := NewTemplatedHelpHandler().WithOutput(&output).
			WithApplicationTemplate("HELP\n").WithCommandTemplate("HELP\n").Create()
		require.NoError(t, err)
		response, err := NewRespondErrorHandler().WithOutput(&output).WithHelpHandler(help).Create()
		require.NoError(t, err)
		app, err := NewSimpleApplication().WithName("app").WithHelpHandler(help).WithErrorHandler(response).Create()
		require.NoError(t, err)
		defer func() { assert.NoError(t, app.Close()) }()
		apps = append(apps, app)
	}
	const count = 64
	start := make(chan struct{})
	results := make(chan error, count*2)
	var workers sync.WaitGroup
	for index := range count {
		app := apps[index%len(apps)]
		workers.Go(func() {
			<-start
			invocation, err := app.Parse(nil, nil, "--help")
			if err == nil {
				err = invocation.Execute(nil)
				if closeErr := invocation.Close(); err == nil {
					err = closeErr
				}
			}
			results <- err
		})
		workers.Go(func() {
			<-start
			invocation, err := app.Parse(nil, nil, "--unknown")
			if invocation != nil || err == nil || err.Error() != "unknown flag --unknown" {
				results <- fmt.Errorf("unexpected parse-error result: invocation=%v, error=%v", invocation, err)
				return
			}
			results <- nil
		})
	}
	close(start)
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	text := output.String()
	assert.Equal(t, count*2, strings.Count(text, "HELP\n"))
	assert.Equal(t, count, strings.Count(text, "Error: unknown flag --unknown\n\n"))
	assert.Equal(t, count*len("Error: unknown flag --unknown\n\n")+count*2*len("HELP\n"), len(text))
}
