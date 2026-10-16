package cli

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/require"
)

// Actual open/close system calls are timed; this is not directly comparable to
// data-only parser benchmarks. Create and preflight are outside the interval.
func BenchmarkApplicationFileLifecycle(b *testing.B) {
	for _, typed := range []bool{false, true} {
		name := "StringDefault"
		if typed {
			name = "TypedDefault"
		}
		b.Run(name, func(b *testing.B) {
			builder := NewSimpleApplication("file").
				WithErrorHandlerEnabled(false)
			flag := NewFlag(codec.NewFile().WithAllowSpecial(true), "input")
			if typed {
				flag.WithDefault(codec.FileRequirement{Path: os.DevNull, Mode: codec.FileRead, AllowSpecial: true})
			} else {
				flag.WithDefaultStr(os.DevNull)
			}
			app, err := builder.WithFlag(flag).Create()
			require.NoError(b, err)
			defer func() { require.NoError(b, app.Close()) }()
			benchmarkParse(b, app, context.Background(), nil)
		})
	}
}

func BenchmarkResourceScopeCleanup(b *testing.B) {
	for _, count := range []int{1, 16, 128} {
		b.Run("Callbacks="+strconv.Itoa(count), func(b *testing.B) {
			closed := 0
			release := func() error { closed++; return nil }
			b.ReportAllocs()
			for b.Loop() {
				scope := new(codec.ResourceScope)
				for range count {
					scope.OnClose(release)
				}
				if err := scope.Close(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			require.Equal(b, b.N*count, closed)
		})
	}
}
