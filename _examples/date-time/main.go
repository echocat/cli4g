package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type options struct {
	Day       time.Time   `flag:"day" type:"date" default:"2026-10-14" description:"Date-only value; full timestamps are rejected."`
	At        time.Time   `flag:"at" default:"2026-10-14T12:30:45.123456789Z" description:"Timestamp using the standard DateTime layouts."`
	Until     time.Time   `argument:"until" type:"custom" default:"15.10.2026 18:00" description:"End time as DD.MM.YYYY HH:MM or RFC3339."`
	Reminders []time.Time `flag:"reminder" type:"date" description:"Date to remember (repeatable)."`
}

func main() {
	cli.NewApplication[options]("date-time", "Parse dates, timestamps, and custom date/time layouts.").
		WithCodec(codec.NewDate(), "date").
		WithCodec(codec.NewDateSlice(), "date").
		WithCodec(codec.NewDateTime().WithFormats("02.01.2006 15:04", time.RFC3339Nano), "custom").
		WithHandler(printCalendar).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func printCalendar(_ context.Context, opts options) error {
	reminders := make([]string, len(opts.Reminders))
	for index, reminder := range opts.Reminders {
		reminders[index] = reminder.Format(time.DateOnly)
	}
	_, err := fmt.Printf("day=%s at=%s until=%s reminders=%v\n",
		opts.Day.Format(time.DateOnly), opts.At.Format(time.RFC3339Nano),
		opts.Until.Format(time.RFC3339Nano), reminders)
	return err
}
