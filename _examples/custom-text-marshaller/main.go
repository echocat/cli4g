package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type date time.Time

func (v date) MarshalText() ([]byte, error) {
	return []byte(time.Time(v).Format(time.DateOnly)), nil
}

func (v *date) UnmarshalText(input []byte) error {
	parsed, err := time.Parse(time.DateOnly, string(input))
	if err != nil {
		return codec.NewValidationErrorf("date must be YYYY-MM-DD: %w", err)
	}
	*v = date(parsed)
	return nil
}

type options struct {
	Date date `flag:"date" default:"2026-01-01" description:"Calendar date (YYYY-MM-DD)."`
}

func main() {
	cli.NewApplication[options]().
		WithName("custom-text-marshaller").
		WithDescription("Use a type implementing text marshaling.").
		WithCodec(codec.NewMarshaller[date]()).
		WithHandler(showDate).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showDate(_ context.Context, opts options) error {
	text, err := opts.Date.MarshalText()
	if err != nil {
		return err
	}
	_, err = fmt.Printf("date=%s\n", text)
	return err
}
