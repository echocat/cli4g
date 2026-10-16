package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapFlagSpellingsAndDuplicateKeys(t *testing.T) {
	type options struct{ Labels map[string]string }
	for _, scope := range []string{"application", "command", "inherited"} {
		t.Run(scope, func(t *testing.T) {
			root := NewApplication[options]("app").
				WithHandler(func(context.Context, options) error { return nil }).
				WithHelp(false)
			child := NewCommand[options]("run")
			configure := func(f *FlagRequirement[map[string]string, map[string]string]) {
				f.WithField("Labels").WithAliases("alias").WithShort('L')
			}
			var prefix []string
			if scope == "command" {
				child.WithStringMapFlag("labels", configure)
				prefix = []string{"run"}
			} else {
				root.WithStringMapFlag("labels", configure)
			}
			root.WithCommand(child)
			app := root.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			args := append(slices.Clone(prefix), "--labels", "env=prod")
			if scope == "inherited" {
				args = append(args, "run")
			}
			args = append(args, "--labels=region=eu", "-Lurl=a=b", "--alias=empty=")
			invocation, err := app.Parse(nil, nil, args...)
			require.NoError(t, err)
			_, actual := invocation.Application()
			if scope == "command" {
				command, found := FindCommand[options](app, "run")
				require.True(t, found)
				actual, found = command.OptionsFrom(invocation)
				require.True(t, found)
			}
			assert.Equal(t, map[string]string{"env": "prod", "region": "eu", "url": "a=b", "empty": ""}, actual.Labels)
			require.NoError(t, invocation.Close())
			for _, duplicate := range []string{"--labels=env=prod", "--alias=env=dev", "-Lenv=prod"} {
				args := append(slices.Clone(prefix), "--labels", "env=prod")
				if scope == "inherited" {
					args = append(args, "run")
				}
				invocation, err := app.Parse(nil, nil, append(args, duplicate)...)
				require.ErrorIs(t, err, ErrValidation)
				assert.ErrorContains(t, err, `duplicate map key "env"`)
				assert.Nil(t, invocation)
			}
		})
	}
}

func TestMapTaggedInputsAndPrecedence(t *testing.T) {
	type options struct {
		Labels map[string]string `flag:"labels" short:"L" env:"MAP_LABELS" default:"env=default"`
		Ports  map[string]int32  `argument:"ports"`
	}
	app := NewApplication[options]("app").
		WithHelp(false).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		name     string
		args     []string
		env      string
		expected options
	}{
		{"default", nil, "", options{Labels: map[string]string{"env": "default"}}},
		{"environment", nil, "env=environment", options{Labels: map[string]string{"env": "environment"}}},
		{"cli", []string{"-Lenv=cli", "http=80", "https=443"}, "env=environment", options{Labels: map[string]string{"env": "cli"}, Ports: map[string]int32{"http": 80, "https": 443}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(nil, &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				return test.env, test.env != ""
			}}, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, actual := invocation.Application()
			assert.Equal(t, test.expected, actual)
		})
	}
	for _, args := range [][]string{{"http=80", "http=81"}, {"invalid"}, {"http=invalid"}} {
		_, err := app.Parse(nil, nil, args...)
		require.ErrorIs(t, err, ErrValidation)
	}
	_, err := NewSimpleApplication("app").
		WithStringMapArgument("map").
		WithStringArgument("after").
		Create()
	require.Error(t, err, "variadic map arguments must be last")
}

func TestMapNonStringKeys(t *testing.T) {
	type identifier uint32
	type options struct {
		Names map[identifier]string
		Bools map[bool]int32
	}
	app := NewApplication[options]("app").WithHelp(false).
		WithFlag(NewFlag(codec.NewMap[identifier](codec.NewString()), "names").WithField("Names")).
		WithArgument(NewArgument(codec.NewMap[bool](codec.NewInt32()), "bools").WithField("Bools")).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation := app.MustParse(nil, nil, "--names=42=answer", "--names=7=seven", "true=1", "false=2")
	_, actual := invocation.Application()
	assert.Equal(t, options{Names: map[identifier]string{42: "answer", 7: "seven"}, Bools: map[bool]int32{true: 1, false: 2}}, actual)
	require.NoError(t, invocation.Close())
	for _, args := range [][]string{
		{"--names=42=first", "--names=0x2a=second"},
		{"true=1", "1=2"},
		{"--names=4294967296=overflow"},
	} {
		_, err := app.Parse(nil, nil, args...)
		require.ErrorIs(t, err, ErrValidation)
	}
}

func TestMapTypedDefaultsAreSnapshottedAndInvocationOwned(t *testing.T) {
	type options struct{ Labels map[string][]string }
	defaults := map[string][]string{"a=b": {"one", "two"}, "nil": nil, "empty": {}}
	app := NewApplication[options]("app").
		WithHelp(false).
		WithFlag(NewFlag(codec.NewMap[string](codec.NewStringSlice()), "labels").
			WithField("Labels").
			WithDefault(defaults),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	defaults["a=b"][0] = "changed builder"
	defaults["extra"] = []string{"extra"}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			invocation, err := app.Parse(nil, nil)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, actual := invocation.Application()
			assert.Equal(t, map[string][]string{"a=b": {"one", "two"}, "nil": nil, "empty": {}}, actual.Labels)
			actual.Labels["a=b"][0] = "changed invocation"
			actual.Labels["extra"] = []string{"extra"}
		})
	}
	workers.Wait()
	for _, value := range []map[string]string{nil, {}} {
		app := NewSimpleApplication("app").WithHelp(false).
			WithStringMapFlag("labels", func(f *FlagRequirement[map[string]string, map[string]string]) {
				f.WithDefault(value)
			}).
			MustCreate()
		invocation := app.MustParse(nil, nil)
		_, actual := invocation.Flag("labels")
		assert.Equal(t, value, actual)
		require.NoError(t, invocation.Close())
		require.NoError(t, app.Close())
	}
}

func TestMapHelpDisplaysSortedDefaults(t *testing.T) {
	var output bytes.Buffer
	app := NewSimpleApplication("app").
		WithHelpHandler(testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()).
		WithStringMapFlag("labels", func(f *FlagRequirement[map[string]string, map[string]string]) {
			f.WithDefault(map[string]string{"z": "last", "a": "first"})
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation := app.MustParse(nil, nil, "--help")
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	assert.Contains(t, output.String(), "default: [a=first, z=last]")
	assert.NotContains(t, output.String(), "map+json:")
}

func TestMapFileHelpersOwnResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
	for _, scope := range []string{"application", "command"} {
		for _, kind := range []string{"flag", "argument"} {
			t.Run(scope+"/"+kind, func(t *testing.T) {
				root := NewSimpleApplication("app").
					WithHelp(false)
				child := NewSimpleCommand("run")
				var args []string
				if scope == "command" {
					args = append(args, "run")
					if kind == "flag" {
						child.WithFileMapFlag("files")
					} else {
						child.WithFileMapArgument("files")
					}
					root.WithCommand(child)
				} else if kind == "flag" {
					root.WithFileMapFlag("files")
				} else {
					root.WithFileMapArgument("files")
				}
				if kind == "flag" {
					args = append(args, "--files=input="+path)
				} else {
					args = append(args, "input="+path)
				}
				app := root.MustCreate()
				defer func() { assert.NoError(t, app.Close()) }()
				invocation := app.MustParse(nil, nil, args...)
				var value any
				if kind == "flag" {
					_, value = invocation.Flag("files")
				} else {
					_, value = invocation.Argument(0)
				}
				files := value.(map[string]*os.File)
				require.NotNil(t, files["input"])
				require.NoError(t, invocation.Close())
				_, err := files["input"].Read(make([]byte, 1))
				assert.ErrorIs(t, err, os.ErrClosed)
			})
		}
	}
}

func TestMapConvenienceStandardCodecs(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"Bool", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithBoolMapFlag, (*CommandRequirement[NoOptions]).WithBoolMapFlag, (*ApplicationRequirement[NoOptions]).WithBoolMapArgument, (*CommandRequirement[NoOptions]).WithBoolMapArgument, []string{"a=true", "b=false"}, map[string]bool{"a": true, "b": false})},
		{"String", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithStringMapFlag, (*CommandRequirement[NoOptions]).WithStringMapFlag, (*ApplicationRequirement[NoOptions]).WithStringMapArgument, (*CommandRequirement[NoOptions]).WithStringMapArgument, []string{"a=", "b=two=three"}, map[string]string{"a": "", "b": "two=three"})},
		{"Int32", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt32MapFlag, (*CommandRequirement[NoOptions]).WithInt32MapFlag, (*ApplicationRequirement[NoOptions]).WithInt32MapArgument, (*CommandRequirement[NoOptions]).WithInt32MapArgument, []string{"a=-32", "b=32"}, map[string]int32{"a": -32, "b": 32})},
		{"Uint64", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint64MapFlag, (*CommandRequirement[NoOptions]).WithUint64MapFlag, (*ApplicationRequirement[NoOptions]).WithUint64MapArgument, (*CommandRequirement[NoOptions]).WithUint64MapArgument, []string{"max=18446744073709551615"}, map[string]uint64{"max": 18446744073709551615})},
		{"Float64", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat64MapFlag, (*CommandRequirement[NoOptions]).WithFloat64MapFlag, (*ApplicationRequirement[NoOptions]).WithFloat64MapArgument, (*CommandRequirement[NoOptions]).WithFloat64MapArgument, []string{"a=1.25"}, map[string]float64{"a": 1.25})},
		{"Complex128", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex128MapFlag, (*CommandRequirement[NoOptions]).WithComplex128MapFlag, (*ApplicationRequirement[NoOptions]).WithComplex128MapArgument, (*CommandRequirement[NoOptions]).WithComplex128MapArgument, []string{"a=1+2i"}, map[string]complex128{"a": 1 + 2i})},
		{"Duration", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithDurationMapFlag, (*CommandRequirement[NoOptions]).WithDurationMapFlag, (*ApplicationRequirement[NoOptions]).WithDurationMapArgument, (*CommandRequirement[NoOptions]).WithDurationMapArgument, []string{"wait=250ms"}, map[string]time.Duration{"wait": 250 * time.Millisecond})},
		{"Date", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateMapFlag, (*CommandRequirement[NoOptions]).WithDateMapFlag, (*ApplicationRequirement[NoOptions]).WithDateMapArgument, (*CommandRequirement[NoOptions]).WithDateMapArgument, []string{"start=2026-10-15"}, map[string]time.Time{"start": time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)})},
		{"DateTime", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateTimeMapFlag, (*CommandRequirement[NoOptions]).WithDateTimeMapFlag, (*ApplicationRequirement[NoOptions]).WithDateTimeMapArgument, (*CommandRequirement[NoOptions]).WithDateTimeMapArgument, []string{"start=2026-10-15T12:30:45Z"}, map[string]time.Time{"start": time.Date(2026, time.October, 15, 12, 30, 45, 0, time.UTC)})},
		{"Marshaller", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithMarshallerMapFlag[marshallerConvenienceValue], (*CommandRequirement[NoOptions]).WithMarshallerMapFlag[marshallerConvenienceValue], (*ApplicationRequirement[NoOptions]).WithMarshallerMapArgument[marshallerConvenienceValue], (*CommandRequirement[NoOptions]).WithMarshallerMapArgument[marshallerConvenienceValue], []string{"a=one", "b=two"}, map[string]marshallerConvenienceValue{"a": {Text: "one"}, "b": {Text: "two"}})},
		{"Value", mapConvenienceCase((*ApplicationRequirement[NoOptions]).WithValueMapFlag[flagReceiverValue], (*CommandRequirement[NoOptions]).WithValueMapFlag[flagReceiverValue], (*ApplicationRequirement[NoOptions]).WithValueMapArgument[flagReceiverValue], (*CommandRequirement[NoOptions]).WithValueMapArgument[flagReceiverValue], []string{"a=one", "b=two"}, map[string]flagReceiverValue{"a": {Text: "one", Hidden: 1}, "b": {Text: "two", Hidden: 1}})},
	} {
		t.Run(test.name, test.run)
	}
}

type mapFactoryValue struct {
	text   string
	closes *int
}

func (this *mapFactoryValue) String() string { return this.text }
func (this *mapFactoryValue) Set(input string) error {
	this.text = input
	if input == "invalid" {
		return NewValidationErrorf("invalid map value")
	}
	return nil
}
func (this *mapFactoryValue) MarshalText() ([]byte, error) { return []byte(this.text), nil }
func (this *mapFactoryValue) UnmarshalText(input []byte) error {
	return this.Set(string(input))
}
func (this *mapFactoryValue) Close() error { *this.closes++; return nil }

func TestMapPointerFactoriesAndCleanup(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, kind := range []string{"marshaller-flag", "value-flag", "marshaller-argument", "value-argument"} {
			t.Run(scope+"/"+kind, func(t *testing.T) {
				created, closed := 0, 0
				factory := func() *mapFactoryValue {
					created++
					return &mapFactoryValue{closes: &closed}
				}
				root := NewSimpleApplication("app").
					WithHelp(false)
				child := NewSimpleCommand("run")
				var prefix []string
				if scope == "command" {
					switch kind {
					case "marshaller-flag":
						child.WithMarshallerPMapFlag("values", factory)
					case "value-flag":
						child.WithValuePMapFlag("values", factory)
					case "marshaller-argument":
						child.WithMarshallerPMapArgument("values", factory)
					case "value-argument":
						child.WithValuePMapArgument("values", factory)
					}
					root.WithCommand(child)
					prefix = []string{"run"}
				} else {
					switch kind {
					case "marshaller-flag":
						root.WithMarshallerPMapFlag("values", factory)
					case "value-flag":
						root.WithValuePMapFlag("values", factory)
					case "marshaller-argument":
						root.WithMarshallerPMapArgument("values", factory)
					case "value-argument":
						root.WithValuePMapArgument("values", factory)
					}
				}
				app := root.MustCreate()
				defer func() { assert.NoError(t, app.Close()) }()
				assert.Zero(t, created, "Create must not acquire values")
				isFlag := kind == "marshaller-flag" || kind == "value-flag"
				for _, mode := range []string{"success", "duplicate", "resolve-error"} {
					created, closed = 0, 0
					pairs := []string{"a=one", "b=two"}
					if mode == "duplicate" {
						pairs[1] = "a=two"
					} else if mode == "resolve-error" {
						pairs[1] = "b=invalid"
					}
					args := slices.Clone(prefix)
					for _, pair := range pairs {
						if isFlag {
							pair = "--values=" + pair
						}
						args = append(args, pair)
					}
					invocation, err := app.Parse(nil, nil, args...)
					if mode != "success" {
						require.ErrorIs(t, err, ErrValidation)
						assert.Nil(t, invocation)
						assert.Equal(t, created, closed, "failed parses must release all acquired values")
						if mode == "duplicate" {
							assert.Zero(t, created, "duplicates must fail before resolution")
						} else {
							assert.Equal(t, 2, created)
						}
						continue
					}
					require.NoError(t, err)
					var value any
					if isFlag {
						_, value = invocation.Flag("values")
					} else {
						_, value = invocation.Argument(0)
					}
					values := value.(map[string]*mapFactoryValue)
					assert.Equal(t, "one", values["a"].text)
					assert.Equal(t, "two", values["b"].text)
					assert.NotSame(t, values["a"], values["b"])
					assert.Equal(t, 2, created)
					assert.Zero(t, closed)
					require.NoError(t, invocation.Close())
					require.NoError(t, invocation.Close())
					assert.Equal(t, 2, closed)
				}
			})
		}
	}
}

func mapConvenienceCase[V, R any](
	applicationFlag func(*ApplicationRequirement[NoOptions], string, ...func(*FlagRequirement[V, R])) *ApplicationRequirement[NoOptions],
	commandFlag func(*CommandRequirement[NoOptions], string, ...func(*FlagRequirement[V, R])) *CommandRequirement[NoOptions],
	applicationArgument func(*ApplicationRequirement[NoOptions], string, ...func(*ArgumentRequirement[V, R])) *ApplicationRequirement[NoOptions],
	commandArgument func(*CommandRequirement[NoOptions], string, ...func(*ArgumentRequirement[V, R])) *CommandRequirement[NoOptions],
	inputs []string, expected V,
) func(*testing.T) {
	return func(t *testing.T) {
		t.Run("flag", flagConvenienceCase(applicationFlag, commandFlag, inputs, expected))
		t.Run("argument", argumentConvenienceCase(applicationArgument, commandArgument, inputs, expected))
	}
}
