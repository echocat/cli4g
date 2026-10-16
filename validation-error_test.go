package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationErrorPreservesMessageCauseAndClassification(t *testing.T) {
	cause := &strconv.NumError{Func: "Atoi", Num: "bad", Err: strconv.ErrSyntax}
	err := NewValidationError(cause)
	assert.Equal(t, cause.Error(), err.Error())
	assert.ErrorIs(t, err, ErrValidation)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	var typed *ValidationError
	require.ErrorAs(t, err, &typed)
	assert.Same(t, cause, typed.Err)
	var number *strconv.NumError
	require.ErrorAs(t, err, &number)
	assert.Same(t, cause, number)
	assert.Same(t, err, NewValidationError(err))
	assert.Nil(t, NewValidationError(nil))
	assert.Equal(t, ErrValidation.Error(), (&ValidationError{}).Error())
}

func TestCliValidationAPIUsesCodecIdentity(t *testing.T) {
	assert.Same(t, codec.ErrValidation, ErrValidation)
	cause := io.ErrUnexpectedEOF
	for _, err := range []error{
		NewValidationError(cause), NewValidationErrorf("invalid input: %w", cause),
		codec.NewValidationError(cause), codec.NewValidationErrorf("invalid input: %w", cause),
	} {
		assert.True(t, IsValidationError(err))
		assert.True(t, codec.IsValidationError(err))
		assert.ErrorIs(t, err, cause)
		var cliError *ValidationError
		var codecError *codec.ValidationError
		require.ErrorAs(t, err, &cliError)
		require.ErrorAs(t, err, &codecError)
		assert.Same(t, cliError, codecError)
	}
}

type unmarkedReadCodec struct{ codec.Codec[string, string] }

func (c unmarkedReadCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}
func (unmarkedReadCodec) ParseRequirement(context.Context, codec.Usage, ...string) (string, error) {
	return "", io.ErrUnexpectedEOF
}

func TestDecoderReadFailuresRemainOperationalAcrossInputSources(t *testing.T) {
	for _, source := range []string{"CLI", "environment", "default", "argument"} {
		t.Run(source, func(t *testing.T) {
			decoder := unmarkedReadCodec{codec.NewString()}
			root := NewSimpleApplication("app").
				WithErrorHandlerEnabled(false)
			var args []string
			if source == "argument" {
				root.WithArgument(NewArgument(decoder, "value"))
				args = []string{"input"}
			} else {
				flag := NewFlag(decoder, "value").WithEnvVar("CLI4G_VALUE")
				if source == "default" {
					flag.WithDefaultStr("input")
				} else if source == "CLI" {
					args = []string{"--value=input"}
				}
				root.WithFlag(flag)
			}
			app := root.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return "input", source == "environment" }}
			invocation, err := app.Parse(context.Background(), opts, args...)
			assert.Nil(t, invocation)
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.False(t, IsValidationError(err))
		})
	}
}

type explicitValidatorOptions struct {
	failure error
	closed  *int
}

func (opts *explicitValidatorOptions) Validate(context.Context) error { return opts.failure }
func (opts *explicitValidatorOptions) Close() error                   { *opts.closed++; return nil }

func TestOptionsValidateRequiresExplicitValidationMarker(t *testing.T) {
	for _, cause := range []error{NewValidationErrorf("invalid settings"), io.ErrUnexpectedEOF, errors.New("storage unavailable")} {
		closed, reported := 0, 0
		app := NewApplication[*explicitValidatorOptions]("app").
			WithCreateOptionsHandler(func(context.Context) (*explicitValidatorOptions, error) {
				return &explicitValidatorOptions{failure: cause, closed: &closed}, nil
			}).
			WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error {
				reported++
				assert.Equal(t, 1, closed)
				assert.Equal(t, IsValidationError(cause), IsValidationError(err))
				return nil
			}).
			MustCreate()
		invocation, err := app.Parse(context.Background(), nil)
		assert.Nil(t, invocation)
		assert.ErrorIs(t, err, cause)
		assert.Equal(t, IsValidationError(cause), IsValidationError(err))
		assert.Equal(t, 1, reported)
		require.NoError(t, app.Close())
	}
}

func TestInputFailuresAreClassifiedWithoutLosingCodecErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		cause error
	}{
		{"unknown long", []string{"--unknown"}, nil},
		{"unknown short", []string{"-z"}, nil},
		{"missing value", []string{"--number"}, codec.ErrArgumentRequired},
		{"invalid number", []string{"--number=bad"}, codec.ErrIllegalArgument},
		{"missing required argument", nil, nil},
		{"too many arguments", []string{"first", "extra"}, codec.ErrTooManyArguments},
		{"duplicate scalar", []string{"--number=1", "--number=2", "name"}, nil},
		{"bad help", []string{"--help=bad"}, codec.ErrIllegalArgument},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var reported error
			app, err := NewSimpleApplication("app").WithIntFlag("number").
				WithStringArgument("name", func(a *ArgumentRequirement[string, string]) { a.WithRequired(true) }).
				WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { calls++; reported = err; return nil }).Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, nil, test.args...)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, ErrValidation)
			var input *ValidationError
			require.ErrorAs(t, err, &input)
			assert.Same(t, reported, err)
			assert.Equal(t, 1, calls)
			if test.cause != nil {
				assert.ErrorIs(t, err, test.cause)
			}
			if test.name == "invalid number" {
				var number *strconv.NumError
				require.ErrorAs(t, err, &number)
				assert.ErrorIs(t, err, strconv.ErrSyntax)
			}
		})
	}
}

func TestRequiredArgumentResponseOmitsApplicationAndCommandPrefixes(t *testing.T) {
	for _, command := range []bool{false, true} {
		t.Run(fmt.Sprint(command), func(t *testing.T) {
			var output bytes.Buffer
			target := "simple"
			response, err := NewRespondErrorHandler().WithOutput(&output).
				WithHelpHandler(func(ctx context.Context, invocation InvocationLike) error {
					current, found := ContainerFromContext(ctx)
					assert.True(t, found)
					assert.Equal(t, target, current.Name(), "container context is preserved outside the message")
					assert.Equal(t, target, invocation.HelpInfo().Target.Name())
					return nil
				}).Create()
			require.NoError(t, err)
			root := NewSimpleApplication("simple").
				WithErrorHandler(response)
			var args []string
			if command {
				target = "greet"
				args = []string{"greet"}
				root.WithCommand(NewSimpleCommand("greet").
					WithStringArgument("name", func(a *ArgumentRequirement[string, string]) { a.WithRequired(true) }))
			} else {
				root.WithStringArgument("name", func(a *ArgumentRequirement[string, string]) { a.WithRequired(true) })
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(context.Background(), nil, args...)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, ErrValidation)
			var input *ValidationError
			require.ErrorAs(t, err, &input)
			assert.Equal(t, "Error: required argument name is missing", strings.Split(output.String(), "\n")[0])
		})
	}
}

type inputSetValue string

func (v *inputSetValue) Set(value string) error {
	_, err := strconv.Atoi(value)
	*v = inputSetValue(value)
	return NewValidationError(err)
}
func (v *inputSetValue) String() string { return string(*v) }

type inputUnmarshalValue int

func (v inputUnmarshalValue) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(v))), nil
}
func (v *inputUnmarshalValue) UnmarshalText(input []byte) error {
	parsed, err := strconv.Atoi(string(input))
	*v = inputUnmarshalValue(parsed)
	return NewValidationError(err)
}

func TestUnmarshalContentFailureIsInputDespiteResolvePhase(t *testing.T) {
	app, err := NewSimpleApplication("app").
		WithMarshallerFlag[inputUnmarshalValue]("value").
		WithErrorHandlerEnabled(false).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	_, err = app.Parse(nil, nil, "--value=bad")
	require.ErrorIs(t, err, ErrValidation)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	var number *strconv.NumError
	require.ErrorAs(t, err, &number)
}

type operationalSetValue struct{}

func (*operationalSetValue) Set(string) error {
	return &fs.PathError{Op: "read", Path: "configuration", Err: fs.ErrPermission}
}
func (*operationalSetValue) String() string { return "" }

func TestResolvedValueErrorsUseContentRatherThanPhaseOrSource(t *testing.T) {
	for _, source := range []string{"CLI", "environment", "default"} {
		t.Run(source, func(t *testing.T) {
			flag := NewFlag(mustInputValueCodec(t), "number").
				WithEnvVar("CLI4G_NUMBER")
			if source == "default" {
				flag.WithDefaultStr("bad")
			}
			app, err := NewSimpleApplication("app").
				WithFlag(flag).
				WithErrorHandlerEnabled(false).
				Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			var args []string
			if source == "CLI" {
				args = []string{"--number=bad"}
			}
			opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return "bad", source == "environment" }}
			invocation, err := app.Parse(nil, opts, args...)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, ErrValidation)
			assert.NotErrorIs(t, err, codec.ErrIllegalArgument, "adapters preserve the target's own classification")
			assert.ErrorIs(t, err, strconv.ErrSyntax)
		})
	}
	app, err := NewSimpleApplication("app").
		WithValueFlag[operationalSetValue]("value").
		WithErrorHandlerEnabled(false).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	_, err = app.Parse(nil, nil, "--value=x")
	require.ErrorIs(t, err, fs.ErrPermission)
	assert.NotErrorIs(t, err, ErrValidation, "adapters preserve unmarked technical failures")
	var path *fs.PathError
	require.ErrorAs(t, err, &path)
}

func mustInputValueCodec(t *testing.T) codec.Codec[inputSetValue, []string] {
	t.Helper()
	result, err := codec.NewValue[inputSetValue]()
	require.NoError(t, err)
	return result
}

type classificationCodec struct {
	codec.Codec[string, string]
	cause  error
	closes *int
}

func (c classificationCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}
func (c classificationCodec) Resolve(_ context.Context, scope codec.Scope, value string) (string, error) {
	scope.OnClose(func() error { *c.closes++; return nil })
	return value, c.cause
}

func TestValidationAndOperationalFailuresRemainDistinctAfterCleanup(t *testing.T) {
	for _, stage := range []string{"validate", "validate unmarked", "validate read error", "validate OS error", "initialize", "factory", "resolve", "resolve canceled", "special file"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New("failed " + stage)
			if stage == "validate" {
				cause = NewValidationError(cause)
			} else if stage == "validate read error" {
				cause = io.ErrUnexpectedEOF
			} else if stage == "validate OS error" {
				cause = &fs.PathError{Op: "stat", Path: "x", Err: fs.ErrPermission}
			} else if stage == "resolve canceled" {
				cause = fmt.Errorf("resolve: %w", context.Canceled)
			} else if stage == "special file" {
				cause = codec.ErrSpecialFile
			}
			closed, calls := 0, 0
			decoder := classificationCodec{Codec: codec.NewString(), closes: &closed}
			if strings.HasPrefix(stage, "resolve") || stage == "special file" {
				decoder.cause = cause
			}
			root := NewSimpleApplication("app").
				WithFlag(NewFlag(decoder, "value").
					WithDefault("data"),
				)
			if strings.HasPrefix(stage, "validate") {
				root.WithValidateHandler(func(context.Context, NoOptions) error { return cause })
			} else if stage == "initialize" {
				root.WithInitializeHandler(func(context.Context, NoOptions) error { return cause })
			} else if stage == "factory" {
				root.WithCreateOptionsHandler(func(context.Context) (NoOptions, error) { return NoOptions{}, cause })
			}
			root.WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
				calls++
				assert.True(t, invocation.(*Invocation[NoOptions]).lifecycle.closed.Load())
				if stage != "factory" {
					assert.Equal(t, 1, closed)
				}
				return nil
			})
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, nil)
			assert.Nil(t, invocation)
			require.ErrorIs(t, err, cause)
			if stage == "validate" || stage == "special file" {
				assert.ErrorIs(t, err, ErrValidation)
				assert.NotContains(t, err.Error(), "app:", "input errors omit the container prefix in validation and resolve too")
			} else {
				assert.NotErrorIs(t, err, ErrValidation)
			}
			assert.Equal(t, 1, calls)
		})
	}
}

func TestUnifiedResponderChoosesHelpAndPreservesOutputFailure(t *testing.T) {
	for _, validation := range []bool{false, true} {
		t.Run(fmt.Sprint(validation), func(t *testing.T) {
			var output bytes.Buffer
			helpCalls := 0
			response := NewRespondErrorHandler().WithOutput(&output).WithHelpHandler(func(context.Context, InvocationLike) error {
				helpCalls++
				_, err := io.WriteString(&output, "HELP\n")
				return err
			})
			handler, err := response.Create()
			require.NoError(t, err)
			response.SetOutput(io.Discard)
			response.SetValidationExitCode(7)
			response.SetExecutionExitCode(8)
			cause := errors.New("failure")
			if validation {
				cause = NewValidationError(cause)
			}
			assert.Same(t, cause, handler(nil, nil, cause))
			assert.Contains(t, output.String(), "Error: failure\n")
			assert.Equal(t, validation, helpCalls == 1)
		})
	}
}

func TestParseAndExecuteReportsCleanupButRetainsApplication(t *testing.T) {
	failure := errors.New("close failed")
	calls, closes := 0, 0
	app, err := NewSimpleApplication("app").
		WithCloseHandler(func(NoOptions) error { closes++; return failure }).
		WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
			calls++
			assert.Equal(t, 1, closes)
			assert.True(t, invocation.(*Invocation[NoOptions]).lifecycle.closed.Load())
			assert.NotErrorIs(t, err, ErrValidation)
			return nil
		}).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	assert.ErrorIs(t, app.ParseAndExecute(nil, nil), failure)
	assert.Equal(t, 1, calls)
	assert.False(t, app.definition.closed.Load(), "caller-owned application stays reusable")
}

func TestNestedParseFailureReachesOuterHandlerOnce(t *testing.T) {
	innerCalls, outerCalls, closes := 0, 0, 0
	inner, err := NewSimpleApplication("inner").
		WithErrorHandler(func(context.Context, InvocationLike, error) error { innerCalls++; return nil }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, inner.Close()) }()
	outer, err := NewSimpleApplication("outer").
		WithHandler(func(ctx context.Context, _ NoOptions) error { _, err := inner.Parse(ctx, nil, "--unknown"); return err }).
		WithCloseHandler(func(NoOptions) error { closes++; return nil }).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error {
			outerCalls++
			assert.Equal(t, 1, closes)
			assert.ErrorIs(t, err, ErrValidation)
			return nil
		}).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, outer.Close()) }()
	assert.ErrorIs(t, outer.ParseAndExecute(nil, nil), ErrValidation)
	assert.Zero(t, innerCalls)
	assert.Equal(t, 1, outerCalls)
}

func TestUnifiedErrorDefaultExitCodesByKind(t *testing.T) {
	stage := os.Getenv("CLI4G_ERROR_KIND_EXIT_TEST")
	if stage != "" {
		DefaultErrorHandler = subprocessDefaultErrorHandler
		root := NewSimpleApplication("app").
			WithHelpHandler(func(context.Context, InvocationLike) error {
				_, err := io.WriteString(os.Stderr, "HELP\n")
				return err
			})
		switch stage {
		case "input":
			root.WithIntFlag("number")
			root.MustCreateParseAndExecute(nil, nil, "--number=bad")
		case "validate":
			root.WithValidateHandler(func(context.Context, NoOptions) error { return NewValidationErrorf("bad settings") }).MustCreateParseAndExecute(nil, nil)
		case "validate unmarked":
			root.WithValidateHandler(func(context.Context, NoOptions) error { return io.ErrUnexpectedEOF }).MustCreateParseAndExecute(nil, nil)
		case "initialize":
			root.WithInitializeHandler(func(context.Context, NoOptions) error { return errors.New("service unavailable") }).MustCreateParseAndExecute(nil, nil)
		case "OS":
			root.WithFileFlag("input").MustCreateParseAndExecute(nil, nil, "--input", "cli4g-missing-directory-93876/input")
		case "canceled":
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			root.MustCreateParseAndExecute(ctx, nil)
		case "deadline":
			root.WithHandler(func(context.Context, NoOptions) error { return context.DeadlineExceeded }).
				MustCreateParseAndExecute(nil, nil)
		case "input during execute":
			root.WithStringFlag("default", func(f *FlagRequirement[string, string]) { f.WithDefault("visible") }).
				WithHandler(func(context.Context, NoOptions) error { return NewValidationErrorf("bad execution input") }).
				WithHelpHandler(func(ctx context.Context, invocation InvocationLike) error {
					definition, found := invocation.Root().Flag("default")
					if !found {
						return errors.New("missing metadata")
					}
					_, _, err := definition.FormatDefault(ctx)
					if err != nil {
						return err
					}
					_, err = io.WriteString(os.Stderr, "HELP\n")
					return err
				}).
				MustCreateParseAndExecute(nil, nil)
		}
		return
	}
	for _, stage := range []string{"input", "validate", "validate unmarked", "initialize", "OS", "canceled", "deadline", "input during execute"} {
		t.Run(stage, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestUnifiedErrorDefaultExitCodesByKind$")
			command.Env = append(os.Environ(), "CLI4G_ERROR_KIND_EXIT_TEST="+stage)
			var output bytes.Buffer
			command.Stderr = &output
			err := command.Run()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, output.String())
			validation := stage == "input" || stage == "validate" || stage == "input during execute"
			want := 1
			if validation {
				want = 2
			} else if stage == "canceled" {
				want = 130
			}
			assert.Equal(t, want, exit.ExitCode())
			assert.Equal(t, validation, strings.Contains(output.String(), "HELP\n"))
			assert.NotContains(t, output.String(), "panic:")
			assert.Equal(t, 1, strings.Count(output.String(), "Error: "))
		})
	}
}
