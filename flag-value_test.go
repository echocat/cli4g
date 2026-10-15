package cli

import (
	"context"
	"errors"
	"flag"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type flagReceiverValue struct {
	Text   string
	Hidden int
}

func (value *flagReceiverValue) Set(text string) error { value.Text = text; value.Hidden++; return nil }
func (value *flagReceiverValue) String() string        { return value.Text }

type flagReceiverBool bool

func (value *flagReceiverBool) Set(text string) error {
	parsed, err := strconv.ParseBool(text)
	if err == nil {
		*value = flagReceiverBool(parsed)
	}
	return err
}
func (value *flagReceiverBool) String() string { return strconv.FormatBool(bool(*value)) }
func (*flagReceiverBool) IsBoolFlag() bool     { return true }

type flagReceiverCounter int

func (value *flagReceiverCounter) Set(text string) error {
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return err
	}
	*value += flagReceiverCounter(parsed)
	return nil
}
func (value *flagReceiverCounter) String() string { return strconv.Itoa(int(*value)) }
func (*flagReceiverCounter) IsCumulative() bool   { return true }

type flagReceiverMap map[string]string

func (value *flagReceiverMap) Set(text string) error {
	key, data, found := strings.Cut(text, "=")
	if !found {
		return errors.New("expected key=value")
	}
	if *value == nil {
		*value = make(flagReceiverMap)
	}
	(*value)[key] = data
	return nil
}
func (value *flagReceiverMap) String() string { return "display-only-map" }
func (*flagReceiverMap) IsCumulative() bool   { return true }

var _ flag.Value = (*flagReceiverValue)(nil)

func TestAutomaticFlagValuePointerReceiversBoolAndCumulativeNonSlices(t *testing.T) {
	type options struct {
		Value   flagReceiverValue   `flag:"value"`
		Bool    flagReceiverBool    `flag:"enabled" short:"e"`
		Counter flagReceiverCounter `flag:"count|counter"`
		Map     flagReceiverMap     `flag:"map"`
	}
	app, err := NewApplication[options]().WithName("app").
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=text", "--enabled", "--count=2", "--counter=3", "--map=a=one", "--map=b=two")
	require.NoError(t, err)
	defer func(owned *Invocation[options]) { assert.NoError(t, owned.Close()) }(invocation)
	_, parsed := invocation.Application()
	assert.Equal(t, flagReceiverValue{Text: "text", Hidden: 1}, parsed.Value)
	assert.True(t, bool(parsed.Bool))
	assert.Equal(t, flagReceiverCounter(5), parsed.Counter)
	assert.Equal(t, flagReceiverMap{"a": "one", "b": "two"}, parsed.Map)
	entries := 0
	for range invocation.Flags() {
		entries++
	}
	assert.Equal(t, 4, entries, "each cumulative flag produces one aggregate entry")
	_, aggregate := invocation.Flag("counter")
	assert.Equal(t, flagReceiverCounter(5), aggregate)
	_, err = app.Parse(nil, nil, "--value=first", "--value=second")
	require.ErrorContains(t, err, "supplied more than once")
	invocation, err = app.Parse(nil, nil, "-e=false")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed = invocation.Application()
	assert.False(t, bool(parsed.Bool))
}

func TestCumulativeFlagValueStateIsStrictlyInvocationLocalInParallel(t *testing.T) {
	type options struct {
		Counter flagReceiverCounter `flag:"count"`
		Map     flagReceiverMap     `flag:"map"`
	}
	app, err := NewApplication[*options]().WithName("app").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	var workers sync.WaitGroup
	for n := range 64 {
		workers.Go(func() {
			name := strconv.Itoa(n)
			invocation, err := app.Parse(nil, nil, "--count=2", "--count=3", "--map=first="+name, "--map=second="+name)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, flagReceiverCounter(5), parsed.Counter)
			assert.Equal(t, flagReceiverMap{"first": name, "second": name}, parsed.Map)
			parsed.Map["private"] = name
		})
	}
	workers.Wait()
}

type constructedFlagValue struct {
	Data     map[string]string
	closes   *atomic.Int64
	setErr   error
	failOn   string
	closeErr error
	onSet    func(string)
}

func (value *constructedFlagValue) Set(text string) error {
	value.Data[text] = text
	if value.onSet != nil {
		value.onSet(text)
	}
	if value.failOn == "" || value.failOn == text {
		return value.setErr
	}
	return nil
}
func (value *constructedFlagValue) String() string { return "constructed" }
func (*constructedFlagValue) IsCumulative() bool   { return true }
func (value *constructedFlagValue) Close() error   { value.closes.Add(1); return value.closeErr }

type constructedFlagOptions struct{ Value *constructedFlagValue }

func (options *constructedFlagOptions) Close() error {
	return nil // codec values are borrowed from the invocation's resource scope
}

func TestFlagValueFactoryRunsOncePerCumulativeFlagAndSetFailureIsOwned(t *testing.T) {
	var factories, closes atomic.Int64
	primary := errors.New("set failed")
	cleanup := errors.New("cleanup failed")
	var created *constructedFlagValue
	decoder, err := codec.NewPointerFlagValue(func() (*constructedFlagValue, error) {
		factories.Add(1)
		created = &constructedFlagValue{Data: make(map[string]string), closes: &closes, closeErr: cleanup}
		return created, nil
	})
	require.NoError(t, err)
	app, err := NewApplication[*constructedFlagOptions]().WithName("app").WithFlag(
		NewFlag(decoder).WithName("value").WithField("Value"),
	).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Zero(t, factories.Load(), "Create must not call resource factories")
	invocation, err := app.Parse(nil, nil, "--value=first", "--value=second")
	require.NoError(t, err)
	assert.EqualValues(t, 1, factories.Load())
	_, parsed := invocation.Application()
	assert.Equal(t, map[string]string{"first": "first", "second": "second"}, parsed.Value.Data)
	require.ErrorIs(t, invocation.Close(), cleanup)
	assert.EqualValues(t, 1, closes.Load())
	// First-Set failure is still decoder-owned and cleaned without options ever
	// taking it. A later failure is invocation-owned and must not be double-closed.
	failed, err := codec.NewPointerFlagValue(func() (*constructedFlagValue, error) {
		return &constructedFlagValue{Data: make(map[string]string), closes: &closes, setErr: primary, closeErr: cleanup}, nil
	})
	require.NoError(t, err)
	failureScope := new(codec.ResourceScope)
	_, err = failed.Resolve(context.Background(), failureScope, []string{"bad"})
	err = errors.Join(err, failureScope.Close())
	require.ErrorIs(t, err, primary)
	require.ErrorIs(t, err, cleanup)
	assert.EqualValues(t, 2, closes.Load())
	later, err := codec.NewPointerFlagValue(func() (*constructedFlagValue, error) {
		return &constructedFlagValue{Data: make(map[string]string), closes: &closes, setErr: primary, failOn: "bad", closeErr: cleanup}, nil
	})
	require.NoError(t, err)
	failedApp, err := NewApplication[*constructedFlagOptions]().WithName("failed").WithFlag(
		NewFlag(later).WithName("value").WithField("Value"),
	).WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, failedApp.Close()) }()
	_, err = failedApp.Parse(nil, nil, "--value=first", "--value=bad")
	require.ErrorIs(t, err, primary)
	require.ErrorIs(t, err, cleanup)
	assert.EqualValues(t, 3, closes.Load(), "later Set failure remains scope-owned")
}

func TestFlagValueFactoryErrorsReleaseReturnedPartialObject(t *testing.T) {
	var closes atomic.Int64
	primary, cleanup := errors.New("factory failed"), errors.New("cleanup failed")
	decoder, err := codec.NewPointerFlagValue(func() (*constructedFlagValue, error) {
		return &constructedFlagValue{closes: &closes, closeErr: cleanup}, primary
	})
	require.NoError(t, err)
	scope := new(codec.ResourceScope)
	_, err = decoder.Resolve(context.Background(), scope, []string{"unused"})
	err = errors.Join(err, scope.Close())
	require.ErrorIs(t, err, primary)
	require.ErrorIs(t, err, cleanup)
	assert.EqualValues(t, 1, closes.Load())
}

type clonedFlagValue struct {
	Text   string
	Hidden []string
}

func (value *clonedFlagValue) Set(text string) error { value.Text = text; return nil }
func (value *clonedFlagValue) String() string        { return value.Text }
func (value clonedFlagValue) Clone() (clonedFlagValue, error) {
	value.Hidden = append([]string(nil), value.Hidden...)
	return value, nil
}

func TestFlagValueTypedTokenDefaultsAreIndependent(t *testing.T) {
	decoder, err := codec.NewFlagValue[clonedFlagValue]()
	require.NoError(t, err)
	original := []string{"visible"}
	app, err := NewSimpleApplication().WithName("app").WithFlag(
		NewFlag(decoder).WithName("value").WithDefault(original),
	).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	original[0] = "modified"
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Flag("value")
	assert.Equal(t, clonedFlagValue{Text: "visible"}, parsed)
}

type nonClonedFlagValue struct {
	Text   string
	Hidden []string
}

func (value *nonClonedFlagValue) Set(text string) error { value.Text = text; return nil }
func (value *nonClonedFlagValue) String() string        { return value.Text }

func TestFlagValueRequirementsDoNotUseRuntimeStringOrClone(t *testing.T) {
	decoder, err := codec.NewFlagValue[nonClonedFlagValue]()
	require.NoError(t, err)
	original := []string{"text"}
	clone, err := codec.CloneRequirement(context.Background(), decoder, original)
	require.NoError(t, err)
	clone[0] = "changed"
	assert.Equal(t, "text", original[0])
	plain, err := decoder.FormatRequirement(context.Background(), original)
	require.NoError(t, err)
	assert.Equal(t, original, plain)
}

type overridingFlagValueCodec struct {
	codec.Codec[flagReceiverValue, []string]
	prefix string
}

func (this overridingFlagValueCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (this overridingFlagValueCodec) Resolve(ctx context.Context, scope codec.Scope, args []string) (flagReceiverValue, error) {
	value, err := this.Codec.Resolve(ctx, scope, args)
	value.Text = this.prefix + value.Text
	return value, err
}

func TestFlagValueRegisteredAndNamedOverridesBeatAutomaticFallback(t *testing.T) {
	decoder, err := codec.NewFlagValue[flagReceiverValue]()
	require.NoError(t, err)
	type options struct {
		Value flagReceiverValue `flag:"value"`
	}
	app, err := NewApplication[options]().WithName("app").WithCodec(overridingFlagValueCodec{Codec: decoder, prefix: "explicit:"}).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=input")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "explicit:input", parsed.Value.Text)
	type namedOptions struct {
		Value flagReceiverValue `flag:"value" type:"named"`
	}
	named, err := NewApplication[namedOptions]().WithName("named").WithCodec(overridingFlagValueCodec{Codec: decoder, prefix: "named:"}, "named").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, named.Close()) }()
	other, err := named.Parse(nil, nil, "--value=input")
	require.NoError(t, err)
	defer func() { assert.NoError(t, other.Close()) }()
	_, value := other.Application()
	assert.Equal(t, "named:input", value.Value.Text)
}

type configuredBoolFlag struct{ Value *bool }

func (value *configuredBoolFlag) Set(text string) error {
	parsed, err := strconv.ParseBool(text)
	if err == nil {
		*value.Value = parsed
	}
	return err
}
func (value *configuredBoolFlag) String() string   { return strconv.FormatBool(*value.Value) }
func (value *configuredBoolFlag) IsBoolFlag() bool { return *value.Value }

func TestFlagValueFactoryCanDeclareTraitsWhenZeroProbeIsNotUsable(t *testing.T) {
	factories := 0
	decoder, err := codec.NewPointerFlagValue(func() (*configuredBoolFlag, error) {
		factories++
		return &configuredBoolFlag{Value: new(bool)}, nil
	}, codec.FlagValueConfig[*configuredBoolFlag]{Traits: &codec.FlagValueTraits{BoolFlag: true}})
	require.NoError(t, err)
	app, err := NewSimpleApplication().WithName("app").WithFlag(NewFlag(decoder).WithName("enabled")).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Zero(t, factories)
	invocation, err := app.Parse(nil, nil, "--enabled")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("enabled")
	assert.True(t, *value.(*configuredBoolFlag).Value)
	assert.Equal(t, 1, factories)
	traits := &codec.FlagValueTraits{BoolFlag: true}
	snapshot, err := codec.NewPointerFlagValue(func() (*configuredBoolFlag, error) {
		return &configuredBoolFlag{Value: new(bool)}, nil
	}, codec.FlagValueConfig[*configuredBoolFlag]{Traits: traits})
	require.NoError(t, err)
	traits.BoolFlag = false
	min, max := snapshot.ExpectMinMax(context.Background(), codec.UsageFlag)
	assert.Equal(t, 0, min, "schema traits must be independent of the caller's configuration")
	assert.Equal(t, 1, max)
}

func TestFlagValueCumulativeMapTypedDefaultsUseExplicitDeepCloneAndCLIReplacesThem(t *testing.T) {
	decoder, err := codec.NewFlagValue[flagReceiverMap]()
	require.NoError(t, err)
	type options struct{ Data flagReceiverMap }
	prototype := []string{"default=entry"}
	app, err := NewApplication[options]().WithName("app").WithFlag(NewFlag(decoder).WithName("map").WithField("Data").WithDefault(prototype)).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	prototype[0] = "changed"
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed := invocation.Application()
	assert.Equal(t, flagReceiverMap{"default": "entry"}, parsed.Data)
	invocation, err = app.Parse(nil, nil, "--map=first=one", "--map=second=two")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed = invocation.Application()
	assert.Equal(t, flagReceiverMap{"first": "one", "second": "two"}, parsed.Data)
}

func TestFlagValueExplicitRequirementHasPriorityAndUnknownNameFails(t *testing.T) {
	adapter, err := codec.NewFlagValue[flagReceiverValue]()
	require.NoError(t, err)
	type options struct {
		Value flagReceiverValue `flag:"value"`
	}
	app, err := NewApplication[options]().WithName("app").WithFlag(
		NewFlag(overridingFlagValueCodec{Codec: adapter, prefix: "requirement:"}).WithName("value").WithField("Value"),
	).WithCodec(overridingFlagValueCodec{Codec: adapter, prefix: "registry:"}).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=input")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed := invocation.Application()
	assert.Equal(t, "requirement:input", parsed.Value.Text)
	missing, err := NewApplication[struct {
		Value flagReceiverValue `flag:"value" type:"missing"`
	}]().WithName("missing").Create()
	assert.Nil(t, missing)
	require.ErrorContains(t, err, "name \"missing\"")
}

type bareFlagCounter int

func (value *bareFlagCounter) Set(text string) error {
	enabled, err := strconv.ParseBool(text)
	if err == nil && enabled {
		*value++
	}
	return err
}
func (value *bareFlagCounter) String() string { return strconv.Itoa(int(*value)) }
func (*bareFlagCounter) IsBoolFlag() bool     { return true }
func (*bareFlagCounter) IsCumulative() bool   { return true }

func TestFlagValueBoolCumulativeCounterAndStandardFlagGetterTypes(t *testing.T) {
	type options struct {
		Count    bareFlagCounter   `flag:"verbose" short:"v"`
		Duration flagReceiverValue `argument:"value"`
	}
	app, err := NewApplication[options]().WithName("app").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "-vv", "--verbose=false", "input")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed := invocation.Application()
	assert.Equal(t, bareFlagCounter(2), parsed.Count)
	assert.Equal(t, "input", parsed.Duration.Text)
	definition, found := app.Flag("verbose")
	require.True(t, found)
	assert.True(t, definition.IsBoolFlag())
	assert.True(t, definition.IsCumulative())
	assert.Equal(t, "-v, --verbose", helpFlagLabel(HelpFlagInfo{Definition: definition, MinValues: 0, MaxValues: 1}))
}

type flagReceiverSlice []string

func (value *flagReceiverSlice) Set(text string) error { *value = append(*value, text); return nil }
func (*flagReceiverSlice) String() string              { return "display-only" }
func (*flagReceiverSlice) IsCumulative() bool          { return true }

func TestFlagValueCumulativeSliceRebindsAggregateWithoutAppendingItsPrefixTwice(t *testing.T) {
	type options struct {
		Values flagReceiverSlice `flag:"value" default:"default"`
	}
	app, err := NewApplication[*options]().WithName("app").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	inherited, err := app.Parse(nil, nil)
	require.NoError(t, err)
	require.NoError(t, inherited.Close())
	_, defaults := inherited.Application()
	assert.Equal(t, flagReceiverSlice{"default"}, defaults.Values)
	args := make([]string, 32)
	expected := make(flagReceiverSlice, len(args))
	for i := range args {
		expected[i] = strconv.Itoa(i)
		args[i] = "--value=" + expected[i]
	}
	invocation, err := app.Parse(nil, nil, args...)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed := invocation.Application()
	assert.Equal(t, expected, parsed.Values)
	_, aggregate := invocation.Flag("value")
	assert.Equal(t, expected, aggregate)
	entries := 0
	for range invocation.Flags() {
		entries++
	}
	assert.Equal(t, 1, entries)
}

func TestFlagValueCancellationDuringLaterSetRetainsAggregateForOptionsCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var closes atomic.Int64
	var created *constructedFlagValue
	decoder, err := codec.NewPointerFlagValue(func() (*constructedFlagValue, error) {
		created = &constructedFlagValue{Data: make(map[string]string), closes: &closes, onSet: func(input string) {
			if input == "cancel" {
				cancel()
			}
		}}
		return created, nil
	})
	require.NoError(t, err)
	app, err := NewApplication[*constructedFlagOptions]().WithName("app").
		WithFlag(NewFlag(decoder).WithName("value").WithField("Value")).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(ctx, nil, "--value=first", "--value=cancel")
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, invocation)
	assert.EqualValues(t, 1, closes.Load())
	assert.Equal(t, map[string]string{"first": "first", "cancel": "cancel"}, created.Data)
}

func BenchmarkApplicationParseFlagValue(b *testing.B) {
	for _, count := range []int{1, 16, 128} {
		args := make([]string, count)
		for i := range args {
			args[i] = "--count=1"
		}
		b.Run("Automatic/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct {
				Count flagReceiverCounter `flag:"count"`
			}
			app, err := NewApplication[*options]().WithName("app").Create()
			require.NoError(b, err)
			defer func() { assert.NoError(b, app.Close()) }()
			benchmarkParse(b, app, context.Background(), nil, args...)
		})
		b.Run("Typed/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct{ Count flagReceiverCounter }
			decoder, err := codec.NewFlagValue[flagReceiverCounter]()
			require.NoError(b, err)
			app, err := NewApplication[*options]().WithName("app").WithFlag(NewFlag(decoder).WithName("count").WithField("Count")).Create()
			require.NoError(b, err)
			defer func() { assert.NoError(b, app.Close()) }()
			benchmarkParse(b, app, context.Background(), nil, args...)
		})
		b.Run("PointerFactory/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct{ Count *flagReceiverCounter }
			decoder, err := codec.NewPointerFlagValue(func() (*flagReceiverCounter, error) { return new(flagReceiverCounter), nil })
			require.NoError(b, err)
			app, err := NewApplication[*options]().WithName("app").WithFlag(NewFlag(decoder).WithName("count").WithField("Count")).Create()
			require.NoError(b, err)
			defer func() { assert.NoError(b, app.Close()) }()
			benchmarkParse(b, app, context.Background(), nil, args...)
		})
	}
}
