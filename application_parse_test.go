package cli

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

// callbackCodec observes the actual decode boundaries without relying on timing.
type callbackCodec[T, R any] struct {
	codec.Codec[T, R]
	onClone      func(context.Context)
	onParse      func(context.Context)
	onCloneValue func(context.Context)
}

func (this callbackCodec[T, R]) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this callbackCodec[T, R]) Resolve(ctx context.Context, scope codec.Scope, requirement R) (T, error) {
	if this.onClone != nil {
		this.onClone(ctx)
	}
	return this.Codec.Resolve(ctx, scope, requirement)
}

func (this callbackCodec[T, R]) ParseRequirement(ctx context.Context, usage codec.Usage, values ...string) (R, error) {
	value, err := this.Codec.ParseRequirement(ctx, usage, values...)
	if this.onParse != nil {
		this.onParse(ctx)
	}
	return value, err
}

func (this callbackCodec[T, R]) CloneRequirement(ctx context.Context, value R) (R, error) {
	cloned, err := codec.CloneRequirement(ctx, this.Codec, value)
	if err == nil && this.onCloneValue != nil {
		this.onCloneValue(ctx)
	}
	return cloned, err
}

func (this callbackCodec[T, R]) IsCumulative() bool { return codec.IsCumulative(this.Codec) }

func (this callbackCodec[T, R]) ParseRequirementInto(ctx context.Context, usage codec.Usage, previous R, values ...string) (R, error) {
	value, err := this.Codec.(codec.Cumulative[R]).ParseRequirementInto(ctx, usage, previous, values...)
	if this.onParse != nil {
		this.onParse(ctx)
	}
	return value, err
}

func TestParseRejectsCanceledContextBeforeProcessing(t *testing.T) {
	for _, test := range []struct {
		name    string
		context func() (context.Context, context.CancelFunc)
		err     error
	}{
		{
			name: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			err: context.Canceled,
		},
		{
			name: "expired deadline",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Unix(0, 0))
			},
			err: context.DeadlineExceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clones := &atomic.Int64{}
			app, err := NewSimpleApplication().WithName("app").
				WithFlag(NewFlag(grammarString{min: 1, max: 1, clones: clones}).WithName("value")).
				WithInitializeHandler(func(context.Context, NoOptions) error {
					assert.Fail(t, "initializer must not run for an already canceled context")
					return nil
				}).Create()
			require.NoError(t, err)
			ctx, cancel := test.context()
			defer cancel()
			for _, args := range [][]string{{"--unknown"}, {"--value=data"}} {
				invocation, err := app.Parse(ctx, nil, args...)
				assert.ErrorIs(t, err, test.err)
				assert.Nil(t, invocation)
			}
			assert.Zero(t, clones.Load())
		})
	}
}

func TestParseNormalizesNilContextBeforeDecoding(t *testing.T) {
	var events []string
	app, err := NewSimpleApplication().WithName("app").
		WithFlag(NewFlag(callbackCodec[string, string]{
			Codec: codec.NewString(),
			onClone: func(ctx context.Context) {
				assert.NotNil(t, ctx)
				events = append(events, "clone")
			},
			onParse: func(ctx context.Context) {
				assert.NotNil(t, ctx)
				events = append(events, "decode")
			},
		}).WithName("value")).
		WithInitializeHandler(func(ctx context.Context, _ NoOptions) error {
			assert.NotNil(t, ctx)
			events = append(events, "initialize")
			return nil
		}).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(nil, nil, "--value=data")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	assert.Equal(t, []string{"decode", "clone", "initialize"}, events)
}

type rootOpts struct {
	Verbose bool   `flag:"verbose" short:"v"`
	Root    string `flag:"root" env:"ROOT_TEST" required:"true"`
}

type childOpts struct {
	Value string `flag:"value"`
	Name  string `argument:"name" required:"true"`
	Extra string
}

type constantOneCodec struct{}

func (c *constantOneCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}

func (*constantOneCodec) CanHandleType(_ context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[int](), nil
}

func (*constantOneCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) { return 1, 1 }

func testApplication(t *testing.T) *Application[*rootOpts] {
	t.Helper()
	a := NewApplication[*rootOpts]().
		WithName("app").
		WithCommand(NewCommand[*childOpts]().
			WithName("run", "r").
			WithFlag(NewFlag(codec.NewString()).
				WithName("extra").
				WithField("Extra"),
			),
		).
		MustCreate()
	t.Cleanup(func() { assert.NoError(t, a.Close()) })
	return a
}

func TestParseMixedAndScoped(t *testing.T) {
	a := testApplication(t)
	parseOpts := &ParseOpts{LookupEnv: func(_ context.Context, name string) (string, bool) { return "env-value", name == "ROOT_TEST" }}
	i, err := a.Parse(context.Background(), parseOpts, "r", "--value=x", "-v", "--extra", "hello", "Jane")
	require.NoError(t, err)
	defer func() { assert.NoError(t, i.Close()) }()
	_, ro := i.Application()
	root := ro
	assert.True(t, root.Verbose)
	assert.Equal(t, "env-value", root.Root)
	stack := i.Stack()
	require.Len(t, stack, 1)
	assert.Equal(t, []string{"run"}, i.NameStack())
	child := stack[0].Options.(*childOpts)
	assert.Equal(t, "x", child.Value)
	assert.Equal(t, "Jane", child.Name)
	assert.Equal(t, "hello", child.Extra)
	_, value := i.Flag("value")
	assert.Equal(t, "x", value)
	flag, value := i.Flag("verbose")
	assert.NotNil(t, flag)
	assert.Equal(t, true, value)
	argument, value := i.Argument(0)
	assert.NotNil(t, argument)
	assert.Equal(t, "Jane", value)
	argument, _ = i.Argument(-1)
	assert.Nil(t, argument, "negative argument index")
	_, err = a.Parse(context.Background(), parseOpts, "run", "--value", "x")
	require.ErrorContains(t, err, "required argument")
}

func TestConcurrentParsing(t *testing.T) {
	a := testApplication(t)
	var wg sync.WaitGroup
	for j := range 80 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprint(n)
			i, err := a.Parse(context.Background(), nil, "--root", name, "run", name)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, i.Close()) }()
			_, opts := i.Application()
			assert.Equal(t, name, opts.Root)
			stack := i.Stack()
			if assert.Len(t, stack, 1) {
				assert.Equal(t, name, stack[0].Options.(*childOpts).Name)
			}
		}(j)
	}
	wg.Wait()
}

func TestParseChecksRequiredValuesBeforeInitialization(t *testing.T) {
	var events []string
	type childOptions struct {
		Name string `argument:"name" required:"true"`
	}
	app := NewSimpleApplication().
		WithName("app").
		WithInitializeHandler(func(context.Context, NoOptions) error {
			events = append(events, "initialize app")
			return nil
		}).
		WithCloseHandler(func(NoOptions) error {
			events = append(events, "close app")
			return nil
		}).
		WithCommand(NewCommand[*childOptions]().
			WithName("child"),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil, "child")
	require.ErrorContains(t, err, "required argument")
	assert.Nil(t, invocation)
	assert.Equal(t, []string{"close app"}, events)
}

func TestParseRollsBackSuccessfulInitializers(t *testing.T) {
	initErr := errors.New("grandchild initialization failed")
	appCloseErr := errors.New("app close failed")
	childCloseErr := errors.New("child close failed")
	var events []string
	app := NewSimpleApplication().
		WithName("app").
		WithInitializeHandler(func(context.Context, NoOptions) error {
			events = append(events, "initialize app")
			return nil
		}).
		WithCloseHandler(func(NoOptions) error {
			events = append(events, "close app")
			return appCloseErr
		}).
		WithCommand(NewSimpleCommand().
			WithName("child").
			WithInitializeHandler(func(context.Context, NoOptions) error {
				events = append(events, "initialize child")
				return nil
			}).
			WithCloseHandler(func(NoOptions) error {
				events = append(events, "close child")
				return childCloseErr
			}).
			WithCommand(NewCommand[NoOptions]().
				WithName("grandchild").
				WithInitializeHandler(func(context.Context, NoOptions) error {
					events = append(events, "initialize grandchild")
					return initErr
				}).
				WithCloseHandler(func(NoOptions) error {
					events = append(events, "close grandchild")
					return nil
				}),
			),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil, "child", "grandchild")
	require.ErrorIs(t, err, initErr)
	assert.ErrorIs(t, err, childCloseErr)
	assert.ErrorIs(t, err, appCloseErr)
	assert.Nil(t, invocation)
	assert.Equal(t, []string{
		"initialize app",
		"initialize child",
		"initialize grandchild",
		"close grandchild",
		"close child",
		"close app",
	}, events)
}

type benchmarkScalarOptions struct {
	Name    string  `flag:"name"`
	Config  string  `flag:"config"`
	Host    string  `flag:"host"`
	Region  string  `flag:"region"`
	Port    uint16  `flag:"port"`
	Retries uint    `flag:"retries"`
	Timeout int64   `flag:"timeout"`
	Workers int     `flag:"workers"`
	Rate    float64 `flag:"rate"`
	Verbose bool    `flag:"verbose"`
}

type benchmarkLifecycleOptions struct{ benchmarkScalarOptions }

func (*benchmarkLifecycleOptions) Validate(context.Context) error   { return nil }
func (*benchmarkLifecycleOptions) Initialize(context.Context) error { return nil }
func (*benchmarkLifecycleOptions) Close() error                     { return nil }

func benchmarkScalarArgs() []string {
	return []string{
		"--name=server", "--config=config.yaml", "--host=localhost", "--region=eu",
		"--port=8080", "--retries=3", "--timeout=30", "--workers=4", "--rate=1.5", "--verbose",
	}
}

func benchmarkCommandRequirement(depth int) *ApplicationRequirement[*benchmarkScalarOptions] {
	requirement := NewApplication[*benchmarkScalarOptions]().WithName("app")
	if depth > 0 {
		first := NewSimpleCommand().
			WithName("command0")
		current := first
		for i := 1; i < depth; i++ {
			next := NewSimpleCommand().
				WithName("command" + strconv.Itoa(i))
			current.WithCommand(next)
			current = next
		}
		requirement.WithCommand(first)
	}
	return requirement
}

func benchmarkCommandArgs(depth int) []string {
	args := make([]string, 0, depth+10)
	for i := range depth {
		args = append(args, "command"+strconv.Itoa(i))
	}
	// Inherited flags after the leaf exercise parent lookup along the full path.
	return append(args, benchmarkScalarArgs()...)
}

func benchmarkParse[O any](b *testing.B, app *Application[O], ctx context.Context, opts *ParseOpts, args ...string) {
	b.Helper()
	probe, err := app.Parse(ctx, opts, args...)
	require.NoError(b, err)
	require.NoError(b, probe.Close())
	b.ReportAllocs()
	for b.Loop() {
		invocation, err := app.Parse(ctx, opts, args...)
		if err != nil {
			b.Fatal(err)
		}
		if err := invocation.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// Parse benchmarks include mandatory Close; Create and input setup are untimed.
// None of these fixtures depends on the process environment or executes handlers.
func BenchmarkApplicationParse(b *testing.B) {
	ctx := context.Background()
	b.Run("Empty", func(b *testing.B) {
		app, err := NewSimpleApplication().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil)
	})
	b.Run("Scalar/ValueOptions", func(b *testing.B) {
		app, err := NewApplication[benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, benchmarkScalarArgs()...)
	})
	b.Run("Scalar/PointerOptions", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, benchmarkScalarArgs()...)
	})
	b.Run("Scalar/CancelableContext", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		benchmarkParse(b, app, ctx, nil, benchmarkScalarArgs()...)
	})
	for _, count := range []int{1, 16, 128} {
		b.Run("RepeatedSlice/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct {
				Values []uint16 `flag:"value"`
			}
			app, err := NewApplication[*options]().WithName("app").Create()
			require.NoError(b, err)
			args := make([]string, count)
			for i := range count {
				args[i] = "--value=" + strconv.Itoa(i)
			}
			benchmarkParse(b, app, ctx, nil, args...)
		})
		b.Run("RepeatedSliceUnbound/Values="+strconv.Itoa(count), func(b *testing.B) {
			app, err := NewSimpleApplication().WithName("app").
				WithFlag(NewFlag(codec.NewSlice(codec.NewUint16())).WithName("value")).Create()
			require.NoError(b, err)
			args := make([]string, count)
			for i := range args {
				args[i] = "--value=" + strconv.Itoa(i)
			}
			benchmarkParse(b, app, ctx, nil, args...)
		})
		b.Run("VariadicArguments/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct {
				Values []uint16 `argument:"values"`
			}
			app, err := NewApplication[*options]().WithName("app").Create()
			require.NoError(b, err)
			args := make([]string, count)
			for i := range count {
				args[i] = strconv.Itoa(i)
			}
			benchmarkParse(b, app, ctx, nil, args...)
		})
	}
	for _, depth := range []int{1, 4, 16} {
		b.Run("CommandPath/Depth="+strconv.Itoa(depth), func(b *testing.B) {
			app, err := benchmarkCommandRequirement(depth).Create()
			require.NoError(b, err)
			benchmarkParse(b, app, ctx, nil, benchmarkCommandArgs(depth)...)
		})
	}
	b.Run("Environment/CustomLookup", func(b *testing.B) {
		type options struct {
			Value string `flag:"value" env:"BENCH_VALUE"`
		}
		app, err := NewApplication[*options]().WithName("app").Create()
		require.NoError(b, err)
		opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
			return "value", true
		}}
		benchmarkParse(b, app, ctx, opts)
	})
	b.Run("OptionalArguments/Omitted=16", func(b *testing.B) {
		requirement := NewSimpleApplication().WithName("app")
		for i := range 16 {
			requirement.WithArgument(NewArgument(codec.NewString()).WithName("value" + strconv.Itoa(i)))
		}
		app, err := requirement.Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil)
	})
	b.Run("Mixed/TwoFlagsAnd128Arguments", func(b *testing.B) {
		type options struct {
			Name    string   `flag:"name"`
			Verbose bool     `flag:"verbose"`
			Values  []uint16 `argument:"values"`
		}
		app, err := NewApplication[*options]().WithName("app").Create()
		require.NoError(b, err)
		args := []string{"--name=server", "--verbose"}
		for i := range 128 {
			args = append(args, strconv.Itoa(i))
		}
		benchmarkParse(b, app, ctx, nil, args...)
	})
	b.Run("Mixed/SeparatedScalarValues", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		args := []string{
			"--name", "server", "--config", "config.yaml", "--host", "localhost", "--region", "eu",
			"--port", "8080", "--retries", "3", "--timeout", "30", "--workers", "4", "--rate", "1.5", "--verbose",
		}
		benchmarkParse(b, app, ctx, nil, args...)
	})
	b.Run("Sparse/OneOfTenFlags", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, "--name=server")
	})
	b.Run("Sparse/RepeatedSliceWithUnusedFlags", func(b *testing.B) {
		type options struct {
			Name    string   `flag:"name"`
			Verbose bool     `flag:"verbose"`
			Values  []uint16 `flag:"value"`
		}
		app, err := NewApplication[*options]().WithName("app").Create()
		require.NoError(b, err)
		args := make([]string, 128)
		for i := range args {
			args[i] = "--value=" + strconv.Itoa(i)
		}
		benchmarkParse(b, app, ctx, nil, args...)
	})
	b.Run("FixedArguments/Values=16", func(b *testing.B) {
		requirement := NewSimpleApplication().WithName("app")
		args := make([]string, 16)
		for i := range args {
			requirement.WithArgument(NewArgument(codec.NewString()).WithName("value" + strconv.Itoa(i)))
			args[i] = strconv.Itoa(i)
		}
		app, err := requirement.Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, args...)
	})
	b.Run("Lifecycle/TypedValueHandlers", func(b *testing.B) {
		app, err := NewApplication[benchmarkScalarOptions]().WithName("app").
			WithValidateHandler(func(context.Context, benchmarkScalarOptions) error { return nil }).
			WithInitializeHandler(func(context.Context, benchmarkScalarOptions) error { return nil }).
			WithCloseHandler(func(benchmarkScalarOptions) error { return nil }).Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, benchmarkScalarArgs()...)
	})
	b.Run("Lifecycle/OptionsMethods", func(b *testing.B) {
		app, err := NewApplication[benchmarkLifecycleOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParse(b, app, ctx, nil, benchmarkScalarArgs()...)
	})
}

// Two GCs discard both the local and victim caches of sync.Pool. GC and input
// setup are untimed; each measured operation is the first Parse + Close after
// eviction. Use a fixed iteration count (for example -benchtime=30x), because
// duration-based calibration does not account for the untimed GC work.
func benchmarkParseAfterGC[O any](b *testing.B, app *Application[O], opts *ParseOpts, args ...string) {
	b.Helper()
	ctx := context.Background()
	probe, err := app.Parse(ctx, opts, args...)
	require.NoError(b, err)
	require.NoError(b, probe.Close())
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		runtime.GC()
		runtime.GC()
		b.StartTimer()
		invocation, err := app.Parse(ctx, opts, args...)
		if err != nil {
			b.Fatal(err)
		}
		if err := invocation.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// AfterGC is a cold-pool measurement, not a process-start or Create benchmark.
func BenchmarkApplicationParseAfterGC(b *testing.B) {
	b.Run("Empty", func(b *testing.B) {
		app, err := NewSimpleApplication().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParseAfterGC(b, app, nil)
	})
	b.Run("Scalar/ValueOptions", func(b *testing.B) {
		app, err := NewApplication[benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParseAfterGC(b, app, nil, benchmarkScalarArgs()...)
	})
	b.Run("Scalar/PointerOptions", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParseAfterGC(b, app, nil, benchmarkScalarArgs()...)
	})
	for _, count := range []int{16, 128} {
		b.Run("RepeatedSlice/Values="+strconv.Itoa(count), func(b *testing.B) {
			type options struct {
				Values []uint16 `flag:"value"`
			}
			app, err := NewApplication[*options]().WithName("app").Create()
			require.NoError(b, err)
			args := make([]string, count)
			for i := range args {
				args[i] = "--value=" + strconv.Itoa(i)
			}
			benchmarkParseAfterGC(b, app, nil, args...)
		})
	}
	b.Run("CommandPath/Depth=16", func(b *testing.B) {
		app, err := benchmarkCommandRequirement(16).Create()
		require.NoError(b, err)
		benchmarkParseAfterGC(b, app, nil, benchmarkCommandArgs(16)...)
	})
	b.Run("Environment/CustomLookup", func(b *testing.B) {
		type options struct {
			Value string `flag:"value" env:"BENCH_VALUE"`
		}
		app, err := NewApplication[*options]().WithName("app").Create()
		require.NoError(b, err)
		opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
			return "value", true
		}}
		benchmarkParseAfterGC(b, app, opts)
	})
}

func benchmarkParseParallel[O any](b *testing.B, app *Application[O], args ...string) {
	b.Helper()
	ctx := context.Background()
	probe, err := app.Parse(ctx, nil, args...)
	require.NoError(b, err)
	require.NoError(b, probe.Close())
	var failureOnce sync.Once
	var failure error
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation, err := app.Parse(ctx, nil, args...)
			if err == nil {
				err = invocation.Close()
			}
			if err != nil {
				failureOnce.Do(func() { failure = err })
				return
			}
		}
	})
	b.StopTimer()
	if failure != nil {
		b.Fatal(failure)
	}
}

// Parallel ns/op represents aggregate throughput, not one invocation's latency.
func BenchmarkApplicationParseParallel(b *testing.B) {
	b.Run("RepeatedSlice/Values=128", func(b *testing.B) {
		type options struct {
			Values []uint16 `flag:"value"`
		}
		app, err := NewApplication[*options]().WithName("app").Create()
		require.NoError(b, err)
		args := make([]string, 128)
		for i := range args {
			args[i] = "--value=" + strconv.Itoa(i)
		}
		benchmarkParseParallel(b, app, args...)
	})
	b.Run("Scalar/ValueOptions", func(b *testing.B) {
		app, err := NewApplication[benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParseParallel(b, app, benchmarkScalarArgs()...)
	})
	b.Run("Scalar/PointerOptions", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").Create()
		require.NoError(b, err)
		benchmarkParseParallel(b, app, benchmarkScalarArgs()...)
	})
	b.Run("CommandPath/Depth=16", func(b *testing.B) {
		app, err := benchmarkCommandRequirement(16).Create()
		require.NoError(b, err)
		benchmarkParseParallel(b, app, benchmarkCommandArgs(16)...)
	})
}

func benchmarkDefaultRequirement(typed bool) *ApplicationRequirement[*benchmarkScalarOptions] {
	requirement := NewApplication[*benchmarkScalarOptions]().WithName("app")
	if typed {
		return requirement.WithFlag(
			NewFlag(codec.NewString()).WithName("name").WithField("Name").WithDefault("server"),
			NewFlag(codec.NewString()).WithName("config").WithField("Config").WithDefault("config.yaml"),
			NewFlag(codec.NewString()).WithName("host").WithField("Host").WithDefault("localhost"),
			NewFlag(codec.NewString()).WithName("region").WithField("Region").WithDefault("eu"),
			NewFlag(codec.NewUint16()).WithName("port").WithField("Port").WithDefault(8080),
			NewFlag(codec.NewUint()).WithName("retries").WithField("Retries").WithDefault(3),
			NewFlag(codec.NewInt64()).WithName("timeout").WithField("Timeout").WithDefault(30),
			NewFlag(codec.NewInt()).WithName("workers").WithField("Workers").WithDefault(4),
			NewFlag(codec.NewFloat64()).WithName("rate").WithField("Rate").WithDefault(1.5),
			NewFlag(codec.NewBool()).WithName("verbose").WithField("Verbose").WithDefault(true),
		)
	}
	return requirement.WithFlag(
		NewFlag(codec.NewString()).WithName("name").WithField("Name").WithDefaultStr("server"),
		NewFlag(codec.NewString()).WithName("config").WithField("Config").WithDefaultStr("config.yaml"),
		NewFlag(codec.NewString()).WithName("host").WithField("Host").WithDefaultStr("localhost"),
		NewFlag(codec.NewString()).WithName("region").WithField("Region").WithDefaultStr("eu"),
		NewFlag(codec.NewUint16()).WithName("port").WithField("Port").WithDefaultStr("8080"),
		NewFlag(codec.NewUint()).WithName("retries").WithField("Retries").WithDefaultStr("3"),
		NewFlag(codec.NewInt64()).WithName("timeout").WithField("Timeout").WithDefaultStr("30"),
		NewFlag(codec.NewInt()).WithName("workers").WithField("Workers").WithDefaultStr("4"),
		NewFlag(codec.NewFloat64()).WithName("rate").WithField("Rate").WithDefaultStr("1.5"),
		NewFlag(codec.NewBool()).WithName("verbose").WithField("Verbose").WithDefaultStr("true"),
	)
}

// Default fixtures have no CLI tokens, so they are feature-cost measurements,
// not direct performance comparisons with the ten explicit scalar flags.
func BenchmarkApplicationParseDefaults(b *testing.B) {
	for _, typed := range []bool{false, true} {
		name := "String"
		if typed {
			name = "Typed"
		}
		b.Run(name, func(b *testing.B) {
			app, err := benchmarkDefaultRequirement(typed).Create()
			require.NoError(b, err)
			benchmarkParse(b, app, context.Background(), nil)
		})
		b.Run(name+"Overridden", func(b *testing.B) {
			app, err := benchmarkDefaultRequirement(typed).Create()
			require.NoError(b, err)
			benchmarkParse(b, app, context.Background(), nil, benchmarkScalarArgs()...)
		})
	}
	b.Run("Factory", func(b *testing.B) {
		app, err := NewApplication[*benchmarkScalarOptions]().WithName("app").
			WithCreateOptionsHandler(func(context.Context) (*benchmarkScalarOptions, error) {
				return &benchmarkScalarOptions{Name: "server", Workers: 4}, nil
			}).Create()
		require.NoError(b, err)
		benchmarkParse(b, app, context.Background(), nil)
	})
}
