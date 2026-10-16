package main

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/echocat/cli4g/codec"
)

type color struct {
	Red, Green, Blue uint8
}

// The requirement is a packed, resource-free 24-bit RGB value. Resolve turns it
// into the color struct supplied to the handler.
type colorCodec struct{}

var _ codec.Codec[color, uint32] = colorCodec{}

func (colorCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[color](), ctx.Err()
}

func (colorCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) {
	return 1, 1
}

func (c colorCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (uint32, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, c, args...); err != nil {
		return 0, err
	}
	input := args[0]
	if len(input) != 7 || input[0] != '#' {
		return 0, codec.NewValidationErrorf("color must be #RRGGBB")
	}
	value, err := strconv.ParseUint(input[1:], 16, 24)
	if err != nil {
		return 0, codec.NewValidationError("color must be #RRGGBB", err)
	}
	return uint32(value), nil
}

func (colorCodec) FormatRequirement(ctx context.Context, value uint32) ([]string, error) {
	if err := validateColor(ctx, value); err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("#%06X", value)}, nil
}

func (c colorCodec) DisplayRequirement(ctx context.Context, value uint32) ([]string, error) {
	return c.FormatRequirement(ctx, value)
}

func (colorCodec) Resolve(ctx context.Context, _ codec.Scope, value uint32) (color, error) {
	if err := validateColor(ctx, value); err != nil {
		return color{}, err
	}
	return color{Red: uint8(value >> 16), Green: uint8(value >> 8), Blue: uint8(value)}, nil
}

func (c colorCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}

func validateColor(ctx context.Context, value uint32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if value > 0xFFFFFF {
		return codec.NewValidationErrorf("color exceeds 24 bits")
	}
	return nil
}
