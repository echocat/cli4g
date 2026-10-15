package cli

import (
	"context"
	"io"

	"github.com/echocat/cli4g/codec"
)

// ContextWithStdio supplies borrowed streams. Nil stderr uses stdout; explicit
// context streams suppress process-stdio fallback, including nil stdin/stdout.
func ContextWithStdio(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) context.Context {
	return codec.ContextWithStdio(ctx, stdin, stdout, stderr)
}

// StdioOfContext returns stdin, stdout and stderr without implicit process fallback.
func StdioOfContext(ctx context.Context) (io.Reader, io.Writer, io.Writer) {
	return codec.StdioOfContext(ctx)
}

// StdinOfContext returns the operation's stdin without implicit fallback.
func StdinOfContext(ctx context.Context) io.Reader {
	return codec.StdinOfContext(ctx)
}

// StdoutOfContext returns the operation's stdout without implicit fallback.
func StdoutOfContext(ctx context.Context) io.Writer {
	return codec.StdoutOfContext(ctx)
}

// StderrOfContext returns the operation's stderr without implicit fallback.
func StderrOfContext(ctx context.Context) io.Writer {
	return codec.StderrOfContext(ctx)
}

// Print writes to context stdout like fmt.Print. Missing output is a no-op.
func Print(ctx context.Context, args ...any) error {
	return codec.Print(ctx, args...)
}

// Printf writes to context stdout like fmt.Printf. Missing output is a no-op.
func Printf(ctx context.Context, format string, args ...any) error {
	return codec.Printf(ctx, format, args...)
}

// Println writes to context stdout like fmt.Println. Missing output is a no-op.
func Println(ctx context.Context, args ...any) error {
	return codec.Println(ctx, args...)
}

// ErrPrint writes to context stderr like fmt.Print. Missing output is a no-op.
func ErrPrint(ctx context.Context, args ...any) error {
	return codec.ErrPrint(ctx, args...)
}

// ErrPrintf writes to context stderr like fmt.Printf. Missing output is a no-op.
func ErrPrintf(ctx context.Context, format string, args ...any) error {
	return codec.ErrPrintf(ctx, format, args...)
}

// ErrPrintln writes to context stderr like fmt.Println. Missing output is a no-op.
func ErrPrintln(ctx context.Context, args ...any) error {
	return codec.ErrPrintln(ctx, args...)
}
