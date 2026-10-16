package main

import (
	"context"
	"os"
	"time"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

// Embed the standard duration codec and add a constraint on its resolved value.
type minimumDurationCodec struct {
	codec.Codec[time.Duration, time.Duration]
	minimum time.Duration
}

func (c minimumDurationCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c) // Preserve this wrapper, including its Resolve method.
}

func (c minimumDurationCodec) Resolve(ctx context.Context, scope codec.Scope, value time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if value < c.minimum {
		return 0, codec.NewValidationErrorf("duration must be at least %s; got %s", c.minimum, value)
	}
	return c.Codec.Resolve(ctx, scope, value)
}

type options struct {
	Interval time.Duration `flag:"interval" type:"positive" default:"1s" description:"A strictly positive duration."`
	Delay    time.Duration `flag:"delay" type:"nonnegative" default:"0s" description:"A duration that may be zero."`
}

func main() {
	cli.NewApplication[options]("named-codecs", "Select different codecs for the same Go type using type tags.").
		WithCodec(minimumDurationCodec{codec.NewDuration(), time.Nanosecond}, "positive").
		WithCodec(minimumDurationCodec{codec.NewDuration(), 0}, "nonnegative").
		WithHandler(showDurations).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showDurations(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "interval=%s delay=%s\n", opts.Interval, opts.Delay)
}
