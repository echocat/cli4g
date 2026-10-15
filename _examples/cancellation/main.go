package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/echocat/cli4g"
)

type options struct {
	Sleep time.Duration `flag:"sleep" short:"s" default:"1s" description:"Time between work steps (must be positive)."`
}

func (opts options) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Sleep <= 0 {
		return cli.NewValidationErrorf("sleep must be greater than zero")
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cli.NewApplication[options]().
		WithName("cancellation").
		WithDescription("Cancel an endlessly running handler with Ctrl+C.").
		WithHandler(work).
		MustCreateParseAndExecute(ctx, nil, os.Args[1:]...)
}

func work(ctx context.Context, opts options) error {
	ticker := time.NewTicker(opts.Sleep)
	defer ticker.Stop()

	if _, err := fmt.Fprintln(os.Stdout, "Working. Press Ctrl+C to stop."); err != nil {
		return err
	}
	for step := 1; ; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := ctx.Err(); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(os.Stdout, "Step %d\n", step); err != nil {
				return err
			}
		}
	}
}
