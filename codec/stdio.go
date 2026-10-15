package codec

import (
	"context"
	"fmt"
	"io"
	"os"
)

type stdioKey struct{}

type stdio struct {
	stdin          io.Reader
	stdout, stderr io.Writer
}

// ContextWithStdio supplies borrowed stdin, stdout and stderr for this operation.
// The library never closes these streams. Nil stderr uses stdout; use io.Discard
// to suppress stderr explicitly. Explicit streams suppress application fallback
// entirely, including when stdin or stdout is nil.
// A nil context is replaced with context.Background.
func ContextWithStdio(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if stderr == nil {
		stderr = stdout
	}
	return context.WithValue(ctx, stdioKey{}, stdio{stdin, stdout, stderr})
}

// StdioOfContext returns stdin, stdout and stderr, or nil, nil, nil when none are
// supplied. It never falls back to process stdio on its own.
func StdioOfContext(ctx context.Context) (stdin io.Reader, stdout, stderr io.Writer) {
	if ctx != nil {
		if streams, ok := ctx.Value(stdioKey{}).(stdio); ok {
			return streams.stdin, streams.stdout, streams.stderr
		}
	}
	return nil, nil, nil
}

// StdinOfContext returns the operation's stdin without implicit fallback.
func StdinOfContext(ctx context.Context) io.Reader {
	v, _, _ := StdioOfContext(ctx)
	return v
}

// StdoutOfContext returns the operation's stdout without implicit fallback.
func StdoutOfContext(ctx context.Context) io.Writer {
	_, v, _ := StdioOfContext(ctx)
	return v
}

// StderrOfContext returns the operation's stderr without implicit process-stdio
// fallback. ContextWithStdio uses stdout when its stderr argument is nil.
func StderrOfContext(ctx context.Context) io.Writer {
	_, _, v := StdioOfContext(ctx)
	return v
}

// ContextWithFallbackToStdio prepares an operation's context. With fallback
// enabled, absent streams are filled with os.Stdin, os.Stdout and os.Stderr.
// Existing context streams, including explicit nil streams, are always preserved.
// Applications call this at their operation boundaries; standalone codec users
// can opt into the same fallback explicitly. A nil context uses context.Background.
func ContextWithFallbackToStdio(ctx context.Context, fallback bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if fallback {
		if _, ok := ctx.Value(stdioKey{}).(stdio); !ok {
			return ContextWithStdio(ctx, os.Stdin, os.Stdout, os.Stderr)
		}
	}
	return ctx
}

// Print writes to context stdout like fmt.Print. Missing output is a no-op.
func Print(ctx context.Context, args ...any) error {
	return print(ctx, StdoutOfContext, args...)
}

// Printf writes to context stdout like fmt.Printf. Missing output is a no-op.
func Printf(ctx context.Context, format string, args ...any) error {
	return printf(ctx, StdoutOfContext, format, args...)
}

// Println writes to context stdout like fmt.Println. Missing output is a no-op.
func Println(ctx context.Context, args ...any) error {
	return println(ctx, StdoutOfContext, args...)
}

// ErrPrint writes to context stderr like fmt.Print. Missing output is a no-op.
func ErrPrint(ctx context.Context, args ...any) error {
	return print(ctx, StderrOfContext, args...)
}

// ErrPrintf writes to context stderr like fmt.Printf. Missing output is a no-op.
func ErrPrintf(ctx context.Context, format string, args ...any) error {
	return printf(ctx, StderrOfContext, format, args...)
}

// ErrPrintln writes to context stderr like fmt.Println. Missing output is a no-op.
func ErrPrintln(ctx context.Context, args ...any) error {
	return println(ctx, StderrOfContext, args...)
}

func print(ctx context.Context, resolver func(context.Context) io.Writer, args ...any) error {
	if out := resolver(ctx); out != nil {
		_, err := fmt.Fprint(out, args...)
		return err
	}
	return nil
}

func printf(ctx context.Context, resolver func(context.Context) io.Writer, format string, args ...any) error {
	if out := resolver(ctx); out != nil {
		_, err := fmt.Fprintf(out, format, args...)
		return err
	}
	return nil
}

func println(ctx context.Context, resolver func(context.Context) io.Writer, args ...any) error {
	if out := resolver(ctx); out != nil {
		_, err := fmt.Fprintln(out, args...)
		return err
	}
	return nil
}
