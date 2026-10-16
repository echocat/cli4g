package cli

import (
	"context"
	"io"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Parse fixtures include Close; render fixtures reuse a help invocation and
// include Execute only. Create and template compilation are always untimed.
func BenchmarkApplicationHelp(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		name := "Disabled"
		if enabled {
			name = "Enabled"
		}
		b.Run("Empty/"+name, func(b *testing.B) {
			app, err := NewSimpleApplication("app").
				WithHelp(enabled).
				Create()
			require.NoError(b, err)
			benchmarkParse(b, app, context.Background(), nil)
		})
		b.Run("Scalar/"+name, func(b *testing.B) {
			app, err := NewApplication[*benchmarkScalarOptions]("app").
				WithHelp(enabled).
				Create()
			require.NoError(b, err)
			benchmarkParse(b, app, context.Background(), nil, benchmarkScalarArgs()...)
		})
	}
	for _, depth := range []int{0, 16} {
		b.Run("Parse/Depth="+strconv.Itoa(depth), func(b *testing.B) {
			app, err := benchmarkCommandRequirement(depth).Create()
			require.NoError(b, err)
			args := make([]string, depth, depth+1)
			for i := range depth {
				args[i] = "command" + strconv.Itoa(i)
			}
			args = append(args, "--help")
			benchmarkParse(b, app, context.Background(), nil, args...)
		})
	}
	for _, child := range []bool{false, true} {
		name := "Root"
		if child {
			name = "Child"
		}
		b.Run("Render/"+name, func(b *testing.B) {
			handler := testDefaultTemplatedHelpRequirement().
				WithOutput(io.Discard).
				MustCreate()
			app := benchmarkDefaultRequirement(true).
				WithHelpHandler(handler).
				WithCommand(NewSimpleCommand("run", "Run a command.")).
				MustCreate()
			args := []string{"--help"}
			if child {
				args = []string{"run", "--help"}
			}
			invocation := app.MustParse(context.Background(), nil, args...)
			defer func() { require.NoError(b, invocation.Close()) }()
			require.NoError(b, invocation.Execute(context.Background()))
			b.ReportAllocs()
			for b.Loop() {
				if err := invocation.Execute(context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkApplicationHelpParallel(b *testing.B) {
	handler, err := testDefaultTemplatedHelpRequirement().WithOutput(io.Discard).Create()
	require.NoError(b, err)
	app, err := benchmarkDefaultRequirement(true).WithHelpHandler(handler).Create()
	require.NoError(b, err)
	var failureOnce sync.Once
	var failure error
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation, err := app.Parse(context.Background(), nil, "--help")
			if err == nil {
				err = invocation.Execute(context.Background())
				closeErr := invocation.Close()
				if err == nil {
					err = closeErr
				}
			}
			if err != nil {
				failureOnce.Do(func() { failure = err })
				return
			}
		}
	})
	b.StopTimer()
	if failure != nil {
		b.Fatal(failure)
	}
}
