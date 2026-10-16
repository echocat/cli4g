package cli

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resolvedText struct{ Text string }
type separatedTextCodec struct{ codec.Codec[string, string] }

func (c separatedTextCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw[*resolvedText, string](ctx, c)
}
func (separatedTextCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[*resolvedText](), ctx.Err()
}
func (separatedTextCodec) Resolve(ctx context.Context, _ codec.Scope, input string) (*resolvedText, error) {
	if input == "fail" {
		return nil, io.ErrUnexpectedEOF
	}
	return &resolvedText{input}, ctx.Err()
}

func TestFailedParseGettersPublishOnlySuccessfullyResolvedRuntimeValues(t *testing.T) {
	for _, stage := range []string{"decode", "flag resolve", "argument resolve"} {
		t.Run(stage, func(t *testing.T) {
			calls := 0
			decoder := separatedTextCodec{codec.NewString()}
			root := NewSimpleApplication("app").
				WithFlag(NewFlag[*resolvedText, string](decoder, "first").WithShort('f')).
				WithFlag(NewFlag[*resolvedText, string](decoder, "second").WithShort('s')).
				WithIntFlag("number").
				WithArgument(NewArgument[*resolvedText, string](decoder, "first-argument")).
				WithArgument(NewArgument[*resolvedText, string](decoder, "second-argument"))
			root.WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
				calls++
				for _, name := range []string{"first", "second", "number"} {
					definition, value := invocation.Flag(name)
					if stage != "decode" && (name == "first" || stage == "argument resolve" && name == "second") {
						require.NotNil(t, definition)
						assert.IsType(t, &resolvedText{}, value)
					} else {
						assert.Nil(t, definition)
						assert.Nil(t, value)
					}
				}
				for definition, value := range invocation.Flags() {
					assert.IsType(t, &resolvedText{}, value, definition.Name())
				}
				_, first := invocation.ShortFlag('f')
				if stage == "decode" {
					assert.Nil(t, first)
				} else {
					assert.IsType(t, &resolvedText{}, first)
				}
				argument, value := invocation.Argument(0)
				if stage == "argument resolve" {
					require.NotNil(t, argument)
					assert.Equal(t, &resolvedText{"argument"}, value)
				} else {
					assert.Nil(t, argument)
					assert.Nil(t, value)
				}
				argument, value = invocation.Argument(1)
				assert.Nil(t, argument)
				assert.Nil(t, value)
				for _, value := range invocation.Arguments() {
					assert.IsType(t, &resolvedText{}, value)
				}
				return nil
			})
			app := root.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			args := []string{"--first=one", "--second=fail"}
			if stage == "decode" {
				args = []string{"--first=one", "--number=invalid"}
			} else if stage == "argument resolve" {
				args = []string{"--first=one", "--second=two", "argument", "fail"}
			}
			invocation, err := app.Parse(context.Background(), nil, args...)
			require.Error(t, err)
			assert.Nil(t, invocation)
			assert.Equal(t, 1, calls)
		})
	}
}
