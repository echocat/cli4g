package codec_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequirementRoundtripsAndIndependentSlices(t *testing.T) {
	ctx := context.Background()
	for _, source := range codec.Defaults(ctx) {
		min, max := source.ExpectMinMax(ctx, codec.UsageDefault)
		if min == 0 && max == -1 {
			continue
		}
		input := "42"
		if matches, _ := source.CanHandleType(ctx, reflect.TypeFor[bool]()); matches {
			input = "true"
		} else if matches, _ := source.CanHandleType(ctx, reflect.TypeFor[time.Duration]()); matches {
			input = "42s"
		} else if matches, _ := source.CanHandleType(ctx, reflect.TypeFor[time.Time]()); matches {
			input = "2026-10-14T12:30:45.123456789Z"
		}
		requirement, err := source.ParseRequirement(ctx, codec.UsageDefault, input)
		require.NoError(t, err)
		plain, err := source.FormatRequirement(ctx, requirement)
		require.NoError(t, err)
		cloned, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, requirement, cloned)
	}
	source := codec.NewStringSlice()
	for _, original := range [][]string{nil, {}, {"one", "two"}} {
		plain, err := source.FormatRequirement(ctx, original)
		require.NoError(t, err)
		cloned, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, original, cloned)
		if len(cloned) > 0 {
			cloned[0] = "changed"
			assert.Equal(t, "one", original[0])
		}
	}
}

func TestResourceScopeReverseOrderExactlyOnceAndJoinedErrors(t *testing.T) {
	scope := new(codec.ResourceScope)
	var events []int
	first, second := errors.New("first cleanup"), errors.New("second cleanup")
	scope.OnClose(func() error { events = append(events, 1); return first })
	scope.OnClose(func() error { events = append(events, 2); return second })
	var workers sync.WaitGroup
	results := make(chan error, 16)
	for range 16 {
		workers.Go(func() { results <- scope.Close() })
	}
	workers.Wait()
	close(results)
	for err := range results {
		assert.ErrorIs(t, err, first)
		assert.ErrorIs(t, err, second)
	}
	assert.Equal(t, []int{2, 1}, events)
}

type testSetValue struct {
	Text   string
	closes *int
}

func (value *testSetValue) Set(input string) error {
	value.Text = input
	if input == "invalid" {
		return errors.New("invalid input")
	}
	return nil
}
func (value *testSetValue) String() string { return "display only" }
func (value *testSetValue) Close() error   { *value.closes++; return nil }

func TestSetAdapterStoresTokensAndAcquiresOnlyDuringResolve(t *testing.T) {
	created, closed := 0, 0
	source, err := codec.NewValueP(func() *testSetValue { created++; return &testSetValue{closes: &closed} })
	require.NoError(t, err)
	requirement, err := source.ParseRequirement(context.Background(), codec.UsageDefault, "configured")
	require.NoError(t, err)
	plain, err := source.FormatRequirement(context.Background(), requirement)
	require.NoError(t, err)
	assert.Equal(t, []string{"configured"}, plain, "String is not used as serialization")
	assert.Zero(t, created)
	for _, input := range []string{"configured", "invalid"} {
		scope := new(codec.ResourceScope)
		value, err := source.Resolve(context.Background(), scope, []string{input})
		if input == "invalid" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, input, value.Text)
		}
		require.NoError(t, scope.Close())
		require.NoError(t, scope.Close())
	}
	assert.Equal(t, 2, created)
	assert.Equal(t, 2, closed)
}

func TestFileRequirementRoundtripModesAndIndependentAcquisition(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []codec.FileMode{codec.FileRead, codec.FileWrite, codec.FileReadWrite} {
		source := codec.NewFile(mode)
		original := codec.FileRequirement{Path: filepath.Join(t.TempDir(), "file"), Mode: mode, Create: mode != codec.FileRead, Permission: 0640}
		if mode == codec.FileRead {
			require.NoError(t, os.WriteFile(original.Path, []byte("content"), 0600))
		}
		plain, err := source.FormatRequirement(ctx, original)
		require.NoError(t, err)
		cloned, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, original, cloned)
		var files []*os.File
		for range 2 {
			scope := new(codec.ResourceScope)
			file, err := source.Resolve(ctx, scope, cloned)
			require.NoError(t, err)
			files = append(files, file)
			if mode != codec.FileRead {
				_, err = file.Write([]byte("data"))
				require.NoError(t, err)
			}
			require.NoError(t, scope.Close())
			_, err = file.Stat()
			assert.ErrorIs(t, err, os.ErrClosed)
		}
		assert.NotSame(t, files[0], files[1])
	}
}

type observedFileCodec struct {
	codec.FileCodec
	opened *[]*os.File
}

func (c observedFileCodec) ToRaw(ctx context.Context) codec.Codec[any, any] { return codec.Raw(ctx, c) }
func (c observedFileCodec) Resolve(ctx context.Context, scope codec.Scope, requirement codec.FileRequirement) (*os.File, error) {
	file, err := c.FileCodec.Resolve(ctx, scope, requirement)
	if file != nil {
		*c.opened = append(*c.opened, file)
	}
	return file, err
}

func TestFileSliceResolveFailureReleasesSuccessfulPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
	var opened []*os.File
	source := codec.NewSlice(observedFileCodec{codec.NewFile(), &opened})
	requirements, err := source.ParseRequirement(context.Background(), codec.UsageDefault, path, path+"-missing")
	require.NoError(t, err)
	scope := new(codec.ResourceScope)
	defer func() { assert.NoError(t, scope.Close()) }()
	_, err = source.Resolve(context.Background(), scope, requirements)
	require.Error(t, err)
	require.Len(t, opened, 1)
	require.NoError(t, scope.Close())
	_, err = opened[0].Stat()
	assert.ErrorIs(t, err, os.ErrClosed)
}

func TestRequirementsRejectResourcesEvenThroughContainers(t *testing.T) {
	file, err := os.Open(os.DevNull)
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	for _, value := range []any{file, []*os.File{file}, struct{ File *os.File }{file}, func() {}} {
		assert.ErrorIs(t, codec.ValidateRequirement(value), codec.ErrResourceRequirement)
	}
}
