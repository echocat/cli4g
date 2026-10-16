package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

const privateCodecInput = "password-NEVER-ECHO-7b19"

func TestCodecErrorMessagesStayPrivateAcrossInputSources(t *testing.T) {
	for _, source := range []string{"flag", "short flag", "argument", "environment", "default"} {
		t.Run(source, func(t *testing.T) {
			var output bytes.Buffer
			help := testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()
			response := NewRespondErrorHandler().WithOutput(&output).WithHelpHandler(help).MustCreate()
			root := NewSimpleApplication("app").WithHelpHandler(help).WithErrorHandler(response)
			var args []string
			if source == "argument" {
				root.WithArgument(NewArgument(codec.NewUint8(), "password"))
				args = []string{privateCodecInput}
			} else {
				flag := NewFlag(codec.NewUint8(), "password").WithShort('p').WithEnvVar("PASSWORD")
				switch source {
				case "flag":
					args = []string{"--password", privateCodecInput}
				case "short flag":
					args = []string{"-p" + privateCodecInput}
				case "default":
					flag.WithDefaultStr(privateCodecInput)
				}
				root.WithFlag(flag)
			}
			app := root.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				return privateCodecInput, source == "environment"
			}}, args...)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, codec.ErrIllegalArgument)
			assert.ErrorIs(t, err, strconv.ErrSyntax)
			var native *strconv.NumError
			require.ErrorAs(t, err, &native)
			assert.Equal(t, privateCodecInput, native.Num)
			assert.Contains(t, err.Error(), "0 or a positive integer expected")
			for _, text := range []string{err.Error(), output.String()} {
				assert.NotContains(t, text, privateCodecInput)
				assert.NotContains(t, text, "strconv.")
				assert.NotContains(t, text, "invalid syntax")
			}
		})
	}
}

func TestUint8CLIErrorMessagesDescribeRequirements(t *testing.T) {
	app := NewSimpleApplication("app").WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(codec.NewUint8(), "count")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct{ input, message string }{
		{"-3", "flag --count: 0 or a positive integer expected"},
		{"256", "flag --count: value should not be larger than 255"},
	} {
		_, err := app.Parse(nil, nil, "--count", test.input)
		assert.EqualError(t, err, test.message)
	}
}

type untrustedRawCodec struct {
	codec.Codec[any, any]
	cause error
}

func (c untrustedRawCodec) ParseRequirement(context.Context, codec.Usage, ...string) (any, error) {
	return nil, c.cause
}

type untrustedValue string

func (v *untrustedValue) String() string { return string(*v) }
func (v *untrustedValue) Set(input string) error {
	*v = untrustedValue(input)
	_, err := strconv.Atoi(input)
	return err
}

type untrustedText string

func (v untrustedText) MarshalText() ([]byte, error) { return []byte(v), nil }
func (v *untrustedText) UnmarshalText(input []byte) error {
	*v = untrustedText(input)
	_, err := strconv.Atoi(string(input))
	return err
}

func TestUnmarkedCustomCodecAndAdapterFailuresAreMasked(t *testing.T) {
	ctx := context.Background()
	native := &strconv.NumError{Func: "custom", Num: privateCodecInput, Err: strconv.ErrSyntax}
	value, err := codec.NewValue[untrustedValue]()
	require.NoError(t, err)
	pointerValue, err := codec.NewValueP(func() *untrustedValue { return new(untrustedValue) })
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		flag FlagRequirementLike
	}{
		{"raw codec", NewFlag(untrustedRawCodec{Codec: codec.NewString().ToRaw(ctx), cause: native}, "password")},
		{"value", NewFlag(value, "password")},
		{"pointer value", NewFlag(pointerValue, "password")},
		{"text", NewFlag(codec.NewMarshaller[untrustedText](), "password")},
		{"pointer text", NewFlag(codec.NewMarshallerP(func() *untrustedText { return new(untrustedText) }), "password")},
		{"slice text", NewFlag(codec.NewSlice(codec.NewMarshaller[untrustedText]()), "password")},
		{"map text", NewFlag(codec.NewMap[string](codec.NewMarshaller[untrustedText]()), "password")},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			response := NewRespondErrorHandler().WithOutput(&output).MustCreate()
			app := NewSimpleApplication("app").WithFlag(test.flag).WithErrorHandler(response).MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			input := privateCodecInput
			if test.name == "map text" {
				input = privateCodecInput + "=" + privateCodecInput
			}
			_, err := app.Parse(nil, nil, "--password", input)
			require.ErrorIs(t, err, strconv.ErrSyntax)
			assert.NotErrorIs(t, err, ErrValidation, "masking must not reclassify technical failures")
			var original *strconv.NumError
			require.ErrorAs(t, err, &original)
			assert.Equal(t, privateCodecInput, original.Num)
			assert.NotContains(t, err.Error(), privateCodecInput)
			assert.NotContains(t, output.String(), privateCodecInput)
			assert.NotContains(t, output.String(), "strconv.")
		})
	}
}

type untrustedValidationOptions struct {
	Password string `flag:"password"`
}

func (v untrustedValidationOptions) Validate(context.Context) error {
	_, err := strconv.Atoi(v.Password)
	return err
}

func TestUnmarkedOptionsValidationMessagesAreMasked(t *testing.T) {
	app := NewApplication[untrustedValidationOptions]("app").WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Parse(nil, nil, "--password", privateCodecInput)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	assert.NotErrorIs(t, err, ErrValidation)
	assert.EqualError(t, err, "app: validate: options could not be validated")
	assert.NotContains(t, err.Error(), privateCodecInput)
	var native *strconv.NumError
	require.ErrorAs(t, err, &native)
	assert.Equal(t, privateCodecInput, native.Num)
}

func TestAuthorValidationMessageAndCauseArePreserved(t *testing.T) {
	native := &strconv.NumError{Func: "author", Num: privateCodecInput, Err: strconv.ErrSyntax}
	authored := NewValidationError("an account number is required", native)
	app := NewSimpleApplication("app").WithErrorHandlerEnabled(false).
		WithValidateHandler(func(context.Context, NoOptions) error { return authored }).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Parse(nil, nil)
	assert.ErrorIs(t, err, authored)
	var marked *ValidationError
	require.ErrorAs(t, err, &marked)
	assert.Same(t, authored, marked)
	assert.EqualError(t, err, "an account number is required")
	assert.ErrorIs(t, err, native)
	var original *strconv.NumError
	require.ErrorAs(t, err, &original)
	assert.Same(t, native, original)
}

func TestDefaultsContinueToAppearInHelp(t *testing.T) {
	var output bytes.Buffer
	help := testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()
	app := NewSimpleApplication("app").WithHelpHandler(help).
		WithFlag(NewFlag(codec.NewString(), "default").WithDefault("configured-default")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.NoError(t, app.ParseAndExecute(nil, nil, "--help"))
	assert.Contains(t, output.String(), "default: configured-default")
}

func TestDiagnosticCausesAreNotFormattedByOuterWrappers(t *testing.T) {
	first := errors.New("first " + privateCodecInput)
	second := errors.New("second " + privateCodecInput)
	err := NewValidationError("requirements not satisfied", errors.Join(first, second))
	wrapped := fmt.Errorf("flag --password: %w", err)
	assert.EqualError(t, wrapped, "flag --password: requirements not satisfied")
	assert.ErrorIs(t, wrapped, first)
	assert.ErrorIs(t, wrapped, second)
}

type privateCleanupOptions struct {
	Password string `flag:"password"`
}

func (privateCleanupOptions) Validate(context.Context) error {
	return NewValidationError("password does not meet requirements")
}

func (v privateCleanupOptions) Close() error {
	return &strconv.NumError{Func: "cleanup", Num: v.Password, Err: strconv.ErrRange}
}

func TestCleanupDoesNotReintroduceInputIntoValidationError(t *testing.T) {
	app := NewApplication[privateCleanupOptions]("app").WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Parse(nil, nil, "--password", privateCodecInput)
	require.ErrorIs(t, err, ErrValidation)
	assert.ErrorIs(t, err, strconv.ErrRange)
	assert.Contains(t, err.Error(), "password does not meet requirements")
	assert.Contains(t, err.Error(), "options could not be closed")
	assert.NotContains(t, err.Error(), privateCodecInput)
	assert.NotContains(t, err.Error(), "strconv.")
	var original *strconv.NumError
	require.ErrorAs(t, err, &original)
	assert.Equal(t, privateCodecInput, original.Num)
}
