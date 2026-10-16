package main

import (
	"context"
	"os"

	"github.com/echocat/cli4g"
)

type options struct {
	Color color `flag:"color" default:"#336699" description:"RGB color as #RRGGBB."`
}

func main() {
	cli.NewApplication[options]("custom-codec", "Decode a color using a custom codec.").
		WithCodec(colorCodec{}).
		WithHandler(showColor).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showColor(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "red=%d green=%d blue=%d\n", opts.Color.Red, opts.Color.Green, opts.Color.Blue)
}
