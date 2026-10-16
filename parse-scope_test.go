package cli

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type initializationTrace struct {
	events     []string
	options    int
	failOption int
	failure    error
}

type tracedOptions struct {
	number      int
	initialized bool
	ctx         *initializationTrace
}

type initializationTraceKey struct{}

func (this *tracedOptions) Initialize(contextValue context.Context) error {
	ctx := contextValue.Value(initializationTraceKey{}).(*initializationTrace)
	this.ctx = ctx
	ctx.options++
	this.number = ctx.options
	ctx.events = append(ctx.events, fmt.Sprintf("initialize options %d", this.number))
	if this.number == ctx.failOption {
		return ctx.failure
	}
	this.initialized = true
	return nil
}

func (this *tracedOptions) Close() error {
	if this.ctx == nil {
		return nil
	}
	this.ctx.events = append(this.ctx.events, fmt.Sprintf("close options %d", this.number))
	this.initialized = false
	return nil
}

func TestParseCleansOptionsWhenContainerInitializationFails(t *testing.T) {
	failure := errors.New("container initialization failed")
	trace := &initializationTrace{}
	ctx := context.WithValue(context.Background(), initializationTraceKey{}, trace)
	app, err := NewApplication[tracedOptions]("app").
		WithInitializeHandler(func(contextValue context.Context, value tracedOptions) error {
			ctx := contextValue.Value(initializationTraceKey{}).(*initializationTrace)
			assert.True(t, value.initialized, "pointer initializer must update value-typed options")
			ctx.events = append(ctx.events, "initialize container")
			return failure
		}).
		WithCloseHandler(func(options tracedOptions) error {
			options.ctx.events = append(options.ctx.events, "close container")
			return nil
		}).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(ctx, nil)
	require.ErrorIs(t, err, failure)
	assert.Nil(t, invocation)
	assert.Equal(t, []string{
		"initialize options 1",
		"initialize container",
		"close container",
		"close options 1",
	}, trace.events)
}

func TestParseClosesOptionsEvenWhenTheirInitializerFails(t *testing.T) {
	failure := errors.New("options initialization failed")
	trace := &initializationTrace{failOption: 2, failure: failure}
	ctx := context.WithValue(context.Background(), initializationTraceKey{}, trace)
	app := NewApplication[*tracedOptions]("app").
		WithCloseHandler(func(options *tracedOptions) error {
			options.ctx.events = append(options.ctx.events, "close app")
			return nil
		}).
		WithCommand(NewCommand[*tracedOptions]("child").
			WithCloseHandler(func(options *tracedOptions) error {
				options.ctx.events = append(options.ctx.events, "close child")
				return nil
			}),
		).
		MustCreate()
	invocation, err := app.Parse(ctx, nil, "child")
	require.ErrorIs(t, err, failure)
	assert.Nil(t, invocation)
	assert.Equal(t, []string{
		"initialize options 1",
		"initialize options 2",
		"close child",
		"close options 2",
		"close app",
		"close options 1",
	}, trace.events)
}

func TestPointerLifecycleHooksForValueOptions(t *testing.T) {
	trace := &initializationTrace{}
	ctx := context.WithValue(context.Background(), initializationTraceKey{}, trace)
	app, err := NewApplication[tracedOptions]("app").
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(ctx, nil)
	require.NoError(t, err)
	_, options := invocation.Application()
	assert.True(t, options.initialized)
	require.NoError(t, invocation.Close())
	assert.Equal(t, []string{"initialize options 1", "close options 1"}, trace.events)
}

func TestDecodedValueOptionsAreVisibleInEveryTypedHandler(t *testing.T) {
	type options struct {
		Name  string   `flag:"name"`
		Count int      `flag:"count"`
		Args  []string `argument:"args"`
	}
	want := options{Name: "value", Count: 2, Args: []string{"one", "two"}}
	var events []string
	check := func(stage string, value options) error {
		assert.Equal(t, want, value)
		events = append(events, stage)
		return nil
	}
	app, err := NewApplication[options]("app").
		WithValidateHandler(func(_ context.Context, value options) error { return check("validate", value) }).
		WithInitializeHandler(func(_ context.Context, value options) error { return check("initialize", value) }).
		WithHandler(func(_ context.Context, value options) error { return check("execute", value) }).
		WithCloseHandler(func(value options) error { return check("close", value) }).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--name=value", "--count=2", "one", "two")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, want, value)
	require.NoError(t, invocation.Execute(context.Background()))
	require.NoError(t, invocation.Close())
	assert.Equal(t, []string{"validate", "initialize", "execute", "close"}, events)
}

type closeOnlyOptions struct {
	Value string `flag:"value"`
}

func (options *closeOnlyOptions) Close() error {
	options.Value = "closed"
	return nil
}

func TestCloseOnlyLifecycleWithoutInitializationHooks(t *testing.T) {
	handlerCalled := false
	app, err := NewApplication[closeOnlyOptions]("app").
		WithCloseHandler(func(options closeOnlyOptions) error {
			assert.Equal(t, "data", options.Value)
			handlerCalled = true
			return nil
		}).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--value=data")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	assert.False(t, handlerCalled)
	require.NoError(t, invocation.Close())
	assert.True(t, handlerCalled)
	_, options := invocation.Application()
	assert.Equal(t, "closed", options.Value)
}

type ownedTestResource struct {
	name     string
	events   *[]string
	failure  error
	closeErr error
	closes   int
}

func (resource *ownedTestResource) UnmarshalText(text []byte) error {
	resource.name = string(text)
	return nil
}

func (resource *ownedTestResource) MarshalText() ([]byte, error) {
	return []byte(resource.name), nil
}

func (resource *ownedTestResource) Close() error {
	resource.closes++
	*resource.events = append(*resource.events, "close options "+resource.name)
	return resource.closeErr
}

type resourceOptions struct {
	Resource *ownedTestResource `flag:"resource"`
	Count    int                `flag:"count" env:"COUNT"`
	Mode     string             `flag:"mode"`
	ctx      context.Context
}

func (options *resourceOptions) Validate(context.Context) error {
	*options.Resource.events = append(*options.Resource.events, "validate options "+options.Resource.name)
	if options.Mode == "validate" {
		return options.Resource.failure
	}
	return nil
}

func (options *resourceOptions) Initialize(ctx context.Context) error {
	options.ctx = ctx
	*options.Resource.events = append(*options.Resource.events, "initialize options "+options.Resource.name)
	if options.Mode == "initialize" {
		return options.Resource.failure
	}
	return nil
}

func (options *resourceOptions) Close() error {
	return nil // codec resources belong exclusively to the invocation scope
}

func TestParseOwnsDecodedResourcesAcrossLifecycleFailures(t *testing.T) {
	validated := []string{
		"validate options root", "validate container root",
		"validate options child", "validate container child",
	}
	for _, tc := range []struct {
		name   string
		args   []string
		env    bool
		events []string
	}{
		{
			name: "flag decoding",
			args: []string{"--count=invalid"},
		},
		{
			name: "environment decoding",
			env:  true,
		},
		{
			name: "options validation",
			args: []string{"--mode=validate"},
			events: []string{
				"validate options root", "validate container root", "validate options child",
			},
		},
		{
			name:   "container validation",
			args:   []string{"--mode=validate-handler"},
			events: validated,
		},
		{
			name: "options initialization",
			args: []string{"--mode=initialize"},
			events: append(append([]string{}, validated...),
				"initialize options root", "initialize container root", "initialize options child"),
		},
		{
			name: "container initialization",
			args: []string{"--mode=initialize-handler"},
			events: append(append([]string{}, validated...),
				"initialize options root", "initialize container root",
				"initialize options child", "initialize container child"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New("lifecycle failed")
			rootCloseErr := errors.New("root close failed")
			childCloseErr := errors.New("child close failed")
			var events []string
			var resources []*ownedTestResource
			resourceCodec := codec.NewMarshallerP(func() *ownedTestResource {
				closeErr := rootCloseErr
				if len(resources) > 0 {
					closeErr = childCloseErr
				}
				resource := &ownedTestResource{events: &events, failure: failure, closeErr: closeErr}
				resources = append(resources, resource)
				return resource
			})
			validate := func(_ context.Context, options *resourceOptions) error {
				events = append(events, "validate container "+options.Resource.name)
				if options.Mode == "validate-handler" {
					return failure
				}
				return nil
			}
			initialize := func(_ context.Context, options *resourceOptions) error {
				events = append(events, "initialize container "+options.Resource.name)
				if options.Mode == "initialize-handler" {
					return failure
				}
				return nil
			}
			app := NewApplication[*resourceOptions]("app").
				WithCodec(resourceCodec).
				WithFlag(NewFlag(resourceCodec, "root-resource").
					WithField("Resource"),
				).
				WithFlag(NewFlag(codec.NewInt(), "root-count").
					WithField("Count"),
				).
				WithFlag(NewFlag(codec.NewString(), "root-mode").
					WithField("Mode"),
				).
				WithValidateHandler(validate).
				WithInitializeHandler(initialize).
				WithCloseHandler(func(*resourceOptions) error {
					events = append(events, "close container root")
					return nil
				}).
				WithCommand(NewCommand[resourceOptions]("child").
					WithValidateHandler(func(ctx context.Context, options resourceOptions) error {
						return validate(ctx, &options)
					}).
					WithInitializeHandler(func(ctx context.Context, options resourceOptions) error {
						return initialize(ctx, &options)
					}).
					WithCloseHandler(func(resourceOptions) error {
						events = append(events, "close container child")
						return nil
					}),
				).
				MustCreate()
			var opts *ParseOpts
			if tc.env {
				opts = &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
					return "invalid", true
				}}
			}
			args := append([]string{"--root-resource=root", "child", "--resource=child"}, tc.args...)
			invocation, err := app.Parse(context.Background(), opts, args...)
			require.Error(t, err)
			assert.Nil(t, invocation)
			if tc.name == "flag decoding" || tc.env {
				assert.ErrorIs(t, err, codec.ErrIllegalArgument)
				assert.Empty(t, resources, "requirement errors occur before resource acquisition")
				assert.Equal(t, []string{"close container child", "close container root"}, events)
				return
			} else {
				assert.ErrorIs(t, err, failure)
			}
			assert.ErrorIs(t, err, rootCloseErr)
			assert.ErrorIs(t, err, childCloseErr)
			expected := append(append([]string{}, tc.events...), "close container child", "close options child", "close container root", "close options root")
			assert.Equal(t, expected, events)
			require.Len(t, resources, 2)
			for _, resource := range resources {
				assert.Equal(t, 1, resource.closes)
			}
		})
	}
}

type untouchedOptions struct {
	Count int `flag:"count"`
}

var errUntouchedOptionsClosed = errors.New("untouched options closed")

func (*untouchedOptions) Close() error {
	return errUntouchedOptionsClosed
}

func TestParseClosesOptionsBeforeTheirDecodingStarts(t *testing.T) {
	app := NewSimpleApplication("app").
		WithFlag(NewFlag(codec.NewInt(), "value")).
		WithCommand(NewCommand[untouchedOptions]("child").
			WithInitializeHandler(func(context.Context, untouchedOptions) error {
				assert.Fail(t, "initialization must not run after a decoding error")
				return nil
			}),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil, "--value=invalid", "child")
	assert.Nil(t, invocation)
	assert.ErrorIs(t, err, codec.ErrIllegalArgument)
	assert.ErrorIs(t, err, errUntouchedOptionsClosed)
}

type cancellationControl struct {
	stage  string
	cancel context.CancelFunc
	events []string
}

type cancellationControlKey struct{}

type cancellationOptions struct {
	Resource *ownedTestResource `flag:"resource"`
	Count    int                `flag:"count" env:"BIFROEST_CLI_TEST_CANCEL_COUNT"`
}

func (*cancellationOptions) Validate(ctx context.Context) error {
	control := ctx.Value(cancellationControlKey{}).(*cancellationControl)
	control.events = append(control.events, "validate options")
	if control.stage == "options validation" {
		control.cancel()
	}
	return nil
}

func (*cancellationOptions) Initialize(ctx context.Context) error {
	control := ctx.Value(cancellationControlKey{}).(*cancellationControl)
	control.events = append(control.events, "initialize options")
	if control.stage == "options initialization" {
		control.cancel()
	}
	return nil
}

func (options *cancellationOptions) Close() error {
	return nil // acquired codec values are scope-owned
}

func TestParseCancellationClosesRetainedResources(t *testing.T) {
	for _, test := range []struct {
		stage  string
		events []string
	}{
		{stage: "flag decoding"},
		{stage: "argument decoding"},
		{stage: "codec clone", events: []string{"clone abort"}},
		{stage: "environment lookup", events: []string{"lookup env"}},
		{stage: "options validation", events: []string{"validate options"}},
		{stage: "container validation", events: []string{"validate options", "validate container"}},
		{
			stage:  "options initialization",
			events: []string{"validate options", "validate container", "initialize options"},
		},
		{
			stage: "container initialization",
			events: []string{
				"validate options", "validate container", "initialize options", "initialize container",
			},
		},
	} {
		t.Run(test.stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			control := &cancellationControl{stage: test.stage, cancel: cancel}
			ctx = context.WithValue(ctx, cancellationControlKey{}, control)
			closeErr := errors.New("resource close failed")
			resource := &ownedTestResource{events: &control.events, closeErr: closeErr}
			resourceCodec := callbackCodec[*ownedTestResource, string]{
				Codec: codec.NewMarshallerP(func() *ownedTestResource { return resource }),
				onParse: func(context.Context) {
					control.events = append(control.events, "decode resource")
					if test.stage == "flag decoding" || test.stage == "argument decoding" {
						cancel()
					}
				},
			}
			requirement := NewApplication[*cancellationOptions]("app").
				WithCodec(resourceCodec).
				WithFlag(NewFlag(callbackCodec[string, string]{
					Codec: codec.NewString(),
					onClone: func(context.Context) {
						control.events = append(control.events, "clone abort")
						cancel()
					},
				}, "abort")).
				WithValidateHandler(func(context.Context, *cancellationOptions) error {
					control.events = append(control.events, "validate container")
					if test.stage == "container validation" {
						cancel()
					}
					return nil
				}).
				WithInitializeHandler(func(context.Context, *cancellationOptions) error {
					control.events = append(control.events, "initialize container")
					if test.stage == "container initialization" {
						cancel()
					}
					return nil
				}).
				WithCloseHandler(func(*cancellationOptions) error {
					control.events = append(control.events, "close container")
					return nil
				})
			args := []string{"--resource=resource"}
			if test.stage == "argument decoding" {
				requirement.WithArgument(NewArgument(resourceCodec, "resource").
					WithField("Resource"),
				)
				args = []string{"resource"}
			}
			if test.stage == "codec clone" {
				args = append(args, "--abort=value")
			}
			app, err := requirement.Create()
			require.NoError(t, err)
			opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				if test.stage == "environment lookup" {
					control.events = append(control.events, "lookup env")
					cancel()
					return "1", true
				}
				return "", false
			}}
			invocation, err := app.Parse(ctx, opts, args...)
			assert.Nil(t, invocation)
			assert.ErrorIs(t, err, context.Canceled)
			expected := append([]string{"decode resource"}, test.events...)
			expected = append(expected, "close container")
			if test.stage == "flag decoding" || test.stage == "argument decoding" || test.stage == "environment lookup" {
				assert.Zero(t, resource.closes, "no resource acquired during requirement preparation")
				assert.NotErrorIs(t, err, closeErr)
			} else {
				assert.ErrorIs(t, err, closeErr)
				assert.Equal(t, 1, resource.closes)
				expected = append(expected, "close options resource")
			}
			assert.Equal(t, expected, control.events)
		})
	}
}

func TestParseCancellationSkipsLaterScopeInitialization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rootClosed := false
	app := NewSimpleApplication("app").
		WithInitializeHandler(func(context.Context, NoOptions) error {
			cancel()
			return nil
		}).
		WithCloseHandler(func(NoOptions) error { rootClosed = true; return nil }).
		WithCommand(NewCommand[untouchedOptions]("child").
			WithInitializeHandler(func(context.Context, untouchedOptions) error {
				assert.Fail(t, "later scope initializer must not run after cancellation")
				return nil
			}),
		).
		MustCreate()
	invocation, err := app.Parse(ctx, nil, "child")
	assert.Nil(t, invocation)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, err, errUntouchedOptionsClosed)
	assert.True(t, rootClosed)
}

type poolContextOptions struct {
	Name   string `flag:"name" required:"true"`
	Number int    `flag:"number"`
	Ready  bool
	Closes int
	ctx    context.Context
}

func (options *poolContextOptions) Initialize(ctx context.Context) error {
	options.ctx = ctx
	options.Ready = true
	return nil
}

func (options *poolContextOptions) Close() error {
	options.Ready = false
	options.Closes++
	if options.ctx != nil {
		invocation, found := InvocationFromContext(options.ctx)
		if !found {
			return errors.New("missing retained invocation during Close")
		}
		_, current := invocation.(*Invocation[poolContextOptions]).Application()
		if current.Name != options.Name || current.Ready || current.Closes != options.Closes {
			return errors.New("retained context lost the canonical options instance")
		}
	}
	return nil
}

type poolChildOptions struct {
	Label  string `flag:"label"`
	Name   string `argument:"name" required:"true"`
	Closes int
}

func (options *poolChildOptions) Close() error { options.Closes++; return nil }

func TestRetainedInvocationsSurviveScopeReuseAndGC(t *testing.T) {
	failure := errors.New("initialization failed")
	var failedInvocation *Invocation[poolContextOptions]
	app := NewApplication[poolContextOptions]("app").
		WithInitializeHandler(func(ctx context.Context, options poolContextOptions) error {
			if options.Name == "fail" {
				invocation, _ := InvocationFromContext(ctx)
				failedInvocation = invocation.(*Invocation[poolContextOptions])
				return failure
			}
			return nil
		}).
		WithHandler(func(ctx context.Context, options poolContextOptions) error {
			invocation, found := InvocationFromContext(ctx)
			if !found || !options.Ready {
				return errors.New("missing ready invocation during execution")
			}
			_, value := invocation.Flag("name")
			if value != options.Name {
				return errors.New("execution observed another parse's values")
			}
			return nil
		}).
		WithCommand(NewCommand[*poolChildOptions]("child")).
		MustCreate()
	child, found := FindCommand[*poolChildOptions](app, "child")
	require.True(t, found)
	parse := func(name, number string) (*Invocation[poolContextOptions], error) {
		return app.Parse(context.Background(), nil,
			"--name="+name, "--number="+number, "child", "--label="+name, name)
	}
	first, err := parse("first", "1")
	require.NoError(t, err)
	defer func() { assert.NoError(t, first.Close()) }()
	_, firstOptions := first.Application()
	savedContext := firstOptions.ctx
	firstChild, found := child.OptionsFrom(first)
	require.True(t, found)
	result, err := parse("fail", "2")
	require.ErrorIs(t, err, failure)
	assert.Nil(t, result)
	require.NotNil(t, failedInvocation)
	failedChild, found := child.OptionsFrom(failedInvocation)
	require.True(t, found)
	for i := range 80 {
		result, err := parse(fmt.Sprint(i), "3")
		require.NoError(t, err)
		require.NoError(t, result.Execute(context.Background()))
		require.NoError(t, result.Close())
		result, err = parse("decode failure", "invalid")
		require.Error(t, err)
		assert.Nil(t, result)
	}
	runtime.GC() // A pool may discard its cached scratch state at any GC.
	result, err = parse("after GC", "4")
	require.NoError(t, err)
	require.NoError(t, result.Close())
	retained, found := InvocationFromContext(savedContext)
	require.True(t, found)
	assert.Same(t, first, retained)
	_, value := retained.Flag("name")
	assert.Equal(t, "first", value)
	_, current := first.Application()
	assert.Equal(t, "first", current.Name)
	assert.True(t, current.Ready)
	assert.Zero(t, current.Closes)
	assert.Equal(t, "first", firstChild.Label)
	assert.Equal(t, "first", firstChild.Name)
	assert.Zero(t, firstChild.Closes)
	_, failedOptions := failedInvocation.Application()
	assert.Equal(t, "fail", failedOptions.Name)
	assert.False(t, failedOptions.Ready)
	assert.Equal(t, 1, failedOptions.Closes)
	assert.Equal(t, "fail", failedChild.Name)
	assert.Equal(t, 1, failedChild.Closes)
	require.NoError(t, first.Execute(context.Background()))
	require.NoError(t, first.Close())
	for i := range 40 {
		result, err := parse("after close "+fmt.Sprint(i), "5")
		require.NoError(t, err)
		require.NoError(t, result.Close())
	}
	_, current = first.Application()
	assert.Equal(t, "first", current.Name)
	assert.False(t, current.Ready)
	assert.Equal(t, 1, current.Closes)
	assert.Equal(t, 1, firstChild.Closes)
	_, value = first.Flag("name")
	assert.Equal(t, "first", value)
}

func TestConcurrentLiveInvocationsAreIndependentOfPooledScopes(t *testing.T) {
	app, err := NewApplication[poolContextOptions]("app").
		Create()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			name := fmt.Sprint(i)
			first, err := app.Parse(context.Background(), nil, "--name="+name)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, first.Close()) }()
			for j := range 20 {
				other, err := app.Parse(context.Background(), nil, "--name="+name+"/"+fmt.Sprint(j))
				if !assert.NoError(t, err) {
					return
				}
				assert.NoError(t, other.Close())
			}
			_, current := first.Application()
			assert.Equal(t, name, current.Name)
			assert.True(t, current.Ready)
			assert.Zero(t, current.Closes)
			assert.NoError(t, first.Close())
			_, current = first.Application()
			assert.Equal(t, name, current.Name)
			assert.Equal(t, 1, current.Closes)
		})
	}
	wg.Wait()
}

func TestScopeResetDetachesFrameAndClearsCachedIndexes(t *testing.T) {
	for _, count := range []int{2, 129} {
		app, err := NewSimpleApplication("app").
			Create()
		require.NoError(t, err)
		frame := &invocationFrame{container: app, value: app.newOptions()}
		index := make(map[*Flag[any, any]]int, count)
		for i := range count {
			index[&Flag[any, any]{names: []string{fmt.Sprint(i)}}] = i + 1
		}
		scope := &parseScope{
			invocationFrame: frame, schema: &app.containerSchema,
			positions: []string{"caller data"}, provided: index, pooled: true,
		}
		scope.reset()
		assert.Nil(t, scope.invocationFrame)
		assert.Nil(t, scope.schema)
		assert.Nil(t, scope.positions)
		assert.False(t, scope.pooled)
		if count > 128 {
			assert.Nil(t, scope.provided)
		} else {
			assert.Empty(t, index, "cached map must not retain compiled definitions")
			assert.NotNil(t, scope.provided)
		}
		assert.Same(t, app, frame.container, "reset must not change the persistent frame")
		assert.Equal(t, NoOptions{}, frame.currentOptions())
	}
}
