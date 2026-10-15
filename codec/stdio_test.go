package codec_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
)

func TestStdioContextPrecedenceAndFallback(t *testing.T) {
	for _, ctx := range []context.Context{nil, context.Background()} {
		stdin, stdout, stderr := codec.StdioOfContext(ctx)
		assert.Nil(t, stdin)
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
		assert.Nil(t, codec.StdinOfContext(ctx))
		assert.Nil(t, codec.StdoutOfContext(ctx))
		assert.Nil(t, codec.StderrOfContext(ctx))
		prepared := codec.ContextWithFallbackToStdio(ctx, false)
		assert.NotNil(t, prepared)
		assert.Nil(t, codec.StdinOfContext(prepared))
		assert.Nil(t, codec.StdoutOfContext(prepared))
		assert.Nil(t, codec.StderrOfContext(prepared))
		prepared = codec.ContextWithFallbackToStdio(ctx, true)
		assert.Same(t, os.Stdin, codec.StdinOfContext(prepared))
		assert.Same(t, os.Stdout, codec.StdoutOfContext(prepared))
		assert.Same(t, os.Stderr, codec.StderrOfContext(prepared))
		assert.Same(t, prepared, codec.ContextWithFallbackToStdio(prepared, true))
	}
	input, output, diagnostic := strings.NewReader("input"), new(bytes.Buffer), new(bytes.Buffer)
	for _, test := range []struct {
		stdin                      io.Reader
		stdout, stderr, wantStderr io.Writer
	}{
		{input, output, diagnostic, diagnostic},
		{input, output, nil, output},
		{nil, output, nil, output},
		{nil, nil, diagnostic, diagnostic},
		{nil, nil, nil, nil},
		{input, output, io.Discard, io.Discard},
	} {
		ctx := codec.ContextWithStdio(nil, test.stdin, test.stdout, test.stderr)
		for _, fallback := range []bool{false, true} {
			prepared := codec.ContextWithFallbackToStdio(ctx, fallback)
			assert.Same(t, ctx, prepared, "explicit streams are reused without allocation")
			stdin, stdout, stderr := codec.StdioOfContext(prepared)
			assert.Equal(t, test.stdin, stdin)
			assert.Equal(t, test.stdout, stdout)
			assert.Equal(t, test.wantStderr, stderr)
			assert.Equal(t, stderr, codec.StderrOfContext(prepared))
		}
	}
}

func TestStdioContextRetainsCancellationAndValues(t *testing.T) {
	type key struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "retained"))
	defer cancel()
	input, output := strings.NewReader("input"), new(bytes.Buffer)
	ctx := codec.ContextWithStdio(parent, input, output, nil)
	assert.Equal(t, "retained", ctx.Value(key{}))
	cancel()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	stdin, stdout, stderr := codec.StdioOfContext(ctx)
	assert.Same(t, input, stdin)
	assert.Same(t, output, stdout)
	assert.Same(t, output, stderr)
	// Nested streams replace all three, including clearing an inherited stderr.
	ctx = codec.ContextWithStdio(ctx, nil, nil, nil)
	assert.Nil(t, codec.StderrOfContext(codec.ContextWithFallbackToStdio(ctx, true)))
}

func BenchmarkStdioOfContext(b *testing.B) {
	ctx := codec.ContextWithStdio(nil, strings.NewReader("input"), new(bytes.Buffer), nil)
	b.ReportAllocs()
	for b.Loop() {
		codec.StdioOfContext(ctx)
	}
}

func BenchmarkStdioContextExistingFallback(b *testing.B) {
	ctx := codec.ContextWithStdio(nil, strings.NewReader("input"), new(bytes.Buffer), nil)
	b.ReportAllocs()
	for b.Loop() {
		codec.ContextWithFallbackToStdio(ctx, true)
	}
}

type stdioErrorWriter struct{}

func (stdioErrorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestStdioPrintHelpersRouteFormatAndHandleMissingOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		write  func(context.Context) error
		want   string
		stderr bool
	}{
		{"Print", func(ctx context.Context) error { return codec.Print(ctx, "value=", 42) }, "value=42", false},
		{"Printf", func(ctx context.Context) error { return codec.Printf(ctx, "value=%d", 42) }, "value=42", false},
		{"Println", func(ctx context.Context) error { return codec.Println(ctx, "value=", 42) }, "value= 42\n", false},
		{"ErrPrint", func(ctx context.Context) error { return codec.ErrPrint(ctx, "value=", 42) }, "value=42", true},
		{"ErrPrintf", func(ctx context.Context) error { return codec.ErrPrintf(ctx, "value=%d", 42) }, "value=42", true},
		{"ErrPrintln", func(ctx context.Context) error { return codec.ErrPrintln(ctx, "value=", 42) }, "value= 42\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			ctx := codec.ContextWithStdio(nil, nil, &stdout, &stderr)
			assert.NoError(t, test.write(ctx))
			if test.stderr {
				assert.Equal(t, test.want, stderr.String())
				assert.Empty(t, stdout.String())
			} else {
				assert.Equal(t, test.want, stdout.String())
				assert.Empty(t, stderr.String())
			}
			for _, missing := range []context.Context{nil, context.Background(), codec.ContextWithStdio(nil, nil, nil, nil)} {
				assert.NoError(t, test.write(missing))
			}
			ctx = codec.ContextWithStdio(nil, nil, stdioErrorWriter{}, stdioErrorWriter{})
			assert.ErrorIs(t, test.write(ctx), io.ErrClosedPipe)
			stdout.Reset()
			assert.NoError(t, test.write(codec.ContextWithStdio(nil, nil, &stdout, nil)))
			assert.Equal(t, test.want, stdout.String(), "nil stderr shares stdout")
		})
	}
}
