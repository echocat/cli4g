package cli

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// These fixtures measure failed Parse, including cleanup and its error handler.
// Their handlers never exit; response output is io.Discard rather than a terminal.
func BenchmarkApplicationParseError(b *testing.B) {
	for _, respond := range []bool{false, true} {
		name := "KeepError"
		handler := ErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err })
		var help HelpHandler
		if respond {
			name = "RespondWithHelp"
			var err error
			help, err = testDefaultTemplatedHelpRequirement().WithOutput(io.Discard).Create()
			require.NoError(b, err)
			handler, err = NewRespondErrorHandler().WithOutput(io.Discard).Create()
			require.NoError(b, err)
		}
		app, err := NewApplication[*benchmarkScalarOptions]("app").
			WithHelpHandler(help).WithErrorHandler(handler).Create()
		require.NoError(b, err)
		for _, stage := range []string{"Scanner", "Decode"} {
			b.Run(name+"/"+stage, func(b *testing.B) {
				arg := "--unknown"
				if stage == "Decode" {
					arg = "--workers=invalid"
				}
				invocation, err := app.Parse(context.Background(), nil, arg)
				require.Error(b, err)
				require.Nil(b, invocation)
				b.ReportAllocs()
				for b.Loop() {
					invocation, err := app.Parse(context.Background(), nil, arg)
					if err == nil || invocation != nil {
						b.Fatal("failed Parse must return nil invocation and an error")
					}
				}
			})
		}
	}
}
