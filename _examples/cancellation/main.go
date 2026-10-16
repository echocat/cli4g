package main

import (
	"context"
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

	cli.NewApplication[options]("cancellation", "Cancel an endlessly running handler with Ctrl+C.").
		WithHandler(work).
		MustCreateParseAndExecute(ctx, nil, os.Args[1:]...)
}

func work(ctx context.Context, opts options) error {
	ticker := time.NewTicker(opts.Sleep)
	defer ticker.Stop()

	if err := cli.Println(ctx, "Working. Press Ctrl+C to stop."); err != nil {
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
			if err := cli.Printf(ctx, "Step %d\n", step); err != nil {
				return err
			}
		}
	}
}
