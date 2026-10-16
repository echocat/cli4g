package cli

import (
	"context"
	"testing"
)

// Exercise the default selection of a profile-capable definition as well as
// dynamic selections. The cross-library suite intentionally uses no profiles.
func BenchmarkProfileSelection(b *testing.B) {
	type options struct {
		Value string `flag:"value" default:"default"`
		Extra string `flag:"extra" profile:"feature" default:"active"`
	}
	for _, test := range []struct {
		name     string
		profiles []string
		args     []string
	}{
		{"Root/Default", nil, []string{"--value=hello"}},
		{"Root/Active", []string{"feature"}, []string{"--value=hello", "--extra=world"}},
		{"Command/Default", nil, []string{"child", "--value=hello"}},
		{"Command/Active", []string{"feature"}, []string{"child", "--value=hello", "--extra=world"}},
	} {
		b.Run(test.name, func(b *testing.B) {
			app := NewApplication[options]("bench").
				WithHandler(func(context.Context, options) error { return nil }).
				WithCommand(NewSimpleCommand("child")).MustCreate()
			b.Cleanup(func() { _ = app.Close() })
			ctx, opts := context.Background(), &ParseOpts{Profiles: test.profiles}
			b.ReportAllocs()
			for b.Loop() {
				if err := app.ParseAndExecute(ctx, opts, test.args...); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
