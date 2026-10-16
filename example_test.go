package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestDemo(t *testing.T) {
	commandA := NewCommand[testCommandAOpts]("a").WithAliases("aliasForA").
		WithCodec(custom1Codec{}).
		WithCodec(customVariadic1Codec{}).
		WithHandler(func(ctx context.Context, options testCommandAOpts) error {
			assert.Equal(t, "foo", options.SomeString)
			assert.Equal(t, "bar", options.RequiredString)
			assert.Equal(t, "commandArg", options.CommandsArg)
			assert.Equal(t, "argumentFoo", options.SomeStringArg)
			assert.True(t, options.SomeBoolArg)
			assert.Equal(t, custom1{v: "custom1Value"}, options.Custom1Arg)
			assert.Equal(t, []string{"extra"}, options.CustomVariadic1Arg.v)
			invocation, found := InvocationFromContext(ctx)
			require.True(t, found)
			assert.Equal(t, []string{"a"}, invocation.NameStack())
			_, flagValue := invocation.Flag("someString")
			assert.Equal(t, "foo", flagValue)
			return nil
		})
	requirement := NewSimpleApplication("test", "test description").
		WithCommand(
			commandA,
			NewCommand[testCommandBOpts]("b"),
		)
	app, err := requirement.Create()
	require.ErrorContains(t, err, `name "moo"`)
	require.Nil(t, app)

	commandA.WithCodec(customVariadic1Codec{}, "moo")
	app, err = requirement.Create()
	require.NoError(t, err)
	require.NotNil(t, app)

	parseCtx := context.Background()
	iv, err := app.Parse(parseCtx, nil, "a",
		"--someString", "foo",
		"--requiredString=bar",
		"--",
		"commandArg", "argumentFoo", "true", "custom1Value", "extra",
	)
	require.NoError(t, err)

	defer func() { require.NoError(t, iv.Close()) }()
	err = iv.Execute(parseCtx)
	require.NoError(t, err)

	assert.Equal(t, []string{"a"}, iv.NameStack())
	command, found := FindCommand[testCommandAOpts](app, "a")
	require.True(t, found)
	parsed, found := command.OptionsFrom(iv)
	require.True(t, found)
	assert.Equal(t, "commandArg", parsed.CommandsArg)
	assert.Equal(t, "argumentFoo", parsed.SomeStringArg)
	assert.True(t, parsed.SomeBoolArg)
	assert.Equal(t, custom1{v: "custom1Value"}, parsed.Custom1Arg)
	assert.Equal(t, []string{"extra"}, parsed.CustomVariadic1Arg.v)

	iv2, err := app.Parse(parseCtx, nil, "b",
		"--",
		"great",
	)
	require.NoError(t, err)

	defer func() { require.NoError(t, iv2.Close()) }()
	err = iv2.Execute(parseCtx)
	require.NoError(t, err)
}

type testFlagsOptsMixin struct {
	SomeString      string          `flag:"someString" description:"This is just some string." placeholder:"some-string"`
	SomeBool        bool            `flag:"someBool" description:"This is just some bool."`
	RequiredString  string          `flag:"requiredString" required:"true"` // required, default: false
	HiddenString    string          `flag:"hiddenString" visible:"false"`   // visible, default: true
	WithShort       string          `flag:"withShort" short:"s"`
	WithAlias       string          `flag:"withAlias|withAlias2"`
	WithShortAlias  string          `flag:"withShortAlias" short:"S|2"`
	Custom1         custom1         `flag:"custom1"`
	CustomVariadic1 customVariadic1 `flag:"customVariadic1" type:"moo"`
}

type testArgumentsOptsMixin struct {
	SomeStringArg      string          `argument:"someString" description:"This is just some string." placeholder:"some-string" required:"true"`
	SomeBoolArg        bool            `argument:"someBool" description:"This is just some bool."`
	Custom1Arg         custom1         `argument:"custom1"`
	CustomVariadic1Arg customVariadic1 `argument:"customVariadic1"`
}

type testCommandAOpts struct {
	testFlagsOptsMixin

	CommandsFlag string `flag:"commandsFlag"`
	CommandsArg  string `arg:"commandsArg"`
	testArgumentsOptsMixin
}

type testCommandBOpts struct {
	Some string `argument:"some" required:"true"`
}

func (this testCommandBOpts) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch this.Some {
	case "great", "littleBitGreat":
	// Ok
	default:
		return NewValidationErrorf("some is not great")
	}
	return nil
}

func (this testCommandBOpts) Close() error {
	switch this.Some {
	case "great":
	// Ok
	default:
		return errors.New("some is not great")
	}
	return nil
}

type custom1 struct {
	v string
}

type custom1Codec struct{}

func (this custom1Codec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this custom1Codec) CanHandleType(_ context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[custom1](), nil
}

func (this custom1Codec) ExpectMinMax(_ context.Context, _ codec.Usage) (int, int) {
	return 1, 1
}

func (this custom1Codec) Parse(ctx context.Context, usg codec.Usage, args ...string) (custom1, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usg, this, args...); err != nil {
		return custom1{}, err
	}
	return custom1{args[0]}, nil
}

func (this custom1Codec) Format(_ context.Context, v custom1) ([]string, error) {
	return []string{v.v}, nil
}

func (this custom1Codec) Clone(_ context.Context) (codec.Codec[custom1, custom1], error) {
	return this, nil
}
func (this custom1Codec) CloneDirect() codec.Codec[custom1, custom1] { return this }

func (this custom1Codec) CloneValue(ctx context.Context, value custom1) (custom1, error) {
	return codec.CloneRequirement(ctx, this, value)
}

type customVariadic1 struct {
	v []string
}

type customVariadic1Codec struct{}

func (this customVariadic1Codec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this customVariadic1Codec) CanHandleType(_ context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[customVariadic1](), nil
}

func (this customVariadic1Codec) ExpectMinMax(_ context.Context, _ codec.Usage) (int, int) {
	return 0, -1
}

func (this customVariadic1Codec) Parse(ctx context.Context, usg codec.Usage, args ...string) (customVariadic1, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usg, this, args...); err != nil {
		return customVariadic1{}, err
	}
	return customVariadic1{append([]string(nil), args...)}, nil
}

func (this customVariadic1Codec) Format(_ context.Context, v customVariadic1) ([]string, error) {
	return v.v, nil
}

func (this customVariadic1Codec) Clone(_ context.Context) (codec.Codec[customVariadic1, customVariadic1], error) {
	return this, nil
}
func (this customVariadic1Codec) CloneDirect() codec.Codec[customVariadic1, customVariadic1] {
	return this
}

func (this customVariadic1Codec) CloneValue(ctx context.Context, value customVariadic1) (customVariadic1, error) {
	return codec.CloneRequirement(ctx, this, value)
}
