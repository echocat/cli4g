package main

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/echocat/cli4g"
)

type level string

func (v *level) Set(input string) error {
	switch input {
	case "debug", "info", "warn", "error":
		*v = level(input)
		return nil
	default:
		return cli.NewValidationErrorf("level must be debug, info, warn, or error")
	}
}

func (v level) String() string { return string(v) }

type headers map[string]string

func (v *headers) Set(input string) error {
	key, value, found := strings.Cut(input, "=")
	if !found || strings.TrimSpace(key) == "" {
		return cli.NewValidationErrorf("header must be KEY=VALUE")
	}
	if *v == nil {
		*v = make(headers)
	}
	(*v)[key] = value
	return nil
}

func (headers) IsCumulative() bool { return true }

func (v headers) String() string {
	keys := make([]string, 0, len(v))
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+v[key])
	}
	return strings.Join(parts, ", ")
}

type options struct {
	Level   level   `flag:"level" short:"l" default:"info" description:"Log level (debug, info, warn, error)."`
	Headers headers `flag:"header" short:"H" description:"Header as KEY=VALUE; may be repeated."`
}

func main() {
	cli.NewApplication[options]("custom-values", "Use Set/String types without explicit codec registration.").
		WithHandler(showOptions).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showOptions(ctx context.Context, opts options) error {
	return cli.Printf(ctx, "level=%s headers=[%s]\n", opts.Level, opts.Headers)
}
