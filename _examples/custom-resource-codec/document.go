package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/echocat/cli4g/codec"
)

type document struct {
	Reader io.Reader
}

type documentCodec struct{}

var _ codec.Codec[*document, string] = documentCodec{}

func (documentCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[*document](), ctx.Err()
}

func (documentCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) {
	return 1, 1
}

func (c documentCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, c, args...); err != nil {
		return "", err
	}
	if args[0] == "" {
		return "", codec.NewValidationErrorf("document path must not be empty")
	}
	return args[0], nil
}

func (documentCodec) FormatRequirement(ctx context.Context, path string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, codec.NewValidationErrorf("document path must not be empty")
	}
	return []string{path}, nil
}

func (c documentCodec) DisplayRequirement(ctx context.Context, path string) ([]string, error) {
	return c.FormatRequirement(ctx, path)
}

func (documentCodec) Resolve(ctx context.Context, scope codec.Scope, path string) (*document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if scope == nil {
		return nil, fmt.Errorf("%w: document resolution requires a scope", codec.ErrInvalid)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	// Register the acquired handle before another operation can fail. The options
	// borrow the reader; its underlying file is owned by this codec scope.
	scope.OnClose(func() error {
		closeErr := file.Close()
		_, printErr := fmt.Fprintln(os.Stderr, "Close:", path)
		return errors.Join(closeErr, printErr)
	})
	if _, err := fmt.Fprintln(os.Stderr, "Open:", path); err != nil {
		return nil, err
	}
	return &document{Reader: bufio.NewReader(file)}, ctx.Err()
}

func (c documentCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}
