package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type format uint8

const (
	jsonFormat format = iota
	yamlFormat
)

func (value format) MarshalText() ([]byte, error) {
	switch value {
	case jsonFormat:
		return []byte("json"), nil
	case yamlFormat:
		return []byte("yaml"), nil
	default:
		return nil, fmt.Errorf("unknown format %d", value)
	}
}

func (value *format) UnmarshalText(input []byte) error {
	switch string(input) {
	case "json":
		*value = jsonFormat
	case "yaml":
		*value = yamlFormat
	default:
		return codec.NewValidationError("expected one of: json, yaml")
	}
	return nil
}

type level string

const (
	infoLevel  level = "info"
	debugLevel level = "debug"
)

func (value *level) String() string { return string(*value) }
func (value *level) Set(input string) error {
	*value = level(input)
	return nil // The enum wrapper restricts the possible inputs before Set.
}

type options struct {
	Color   string   `flag:"color" enum:"auto|always|never" default:"auto" description:"Color policy."`
	Format  format   `flag:"format" default:"json" description:"Output encoding."`
	Formats []format `flag:"extra-format" description:"Additional encoding (repeatable)."`
	Level   level    `argument:"level" default:"info" description:"Logging level."`
}

func main() {
	formats, err := codec.NewMarshallerEnumVariants(
		codec.EnumVariant[format]{Value: jsonFormat, Description: "Machine-readable JSON."},
		codec.EnumVariant[format]{Value: yamlFormat, Description: "Readable YAML."},
	)
	if err != nil {
		panic(err)
	}
	levels, err := codec.NewValueEnum(infoLevel, debugLevel)
	if err != nil {
		panic(err)
	}
	cli.NewApplication[options]("enums", "Typed enums with automatic choices in help and errors.").
		WithCodec(formats).
		WithCodec(codec.NewSlice(formats)).
		WithCodec(levels).
		WithHandler(showOptions).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func showOptions(ctx context.Context, opts options) error {
	encoded, err := opts.Format.MarshalText()
	if err != nil {
		return err
	}
	return cli.Printf(ctx, "color=%s format=%s level=%s extra=%d\n", opts.Color, encoded, opts.Level, len(opts.Formats))
}
