package codec

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
)

// Scope owns resources acquired by a codec during Resolve.
// Register only codec-acquired resources, never caller-owned values.
type Scope interface {
	// OnClose registers cleanup immediately after acquisition, including partial failures.
	OnClose(func() error)
}

func ownValue[T any](scope Scope, value T) {
	if isNil(value) {
		return
	}
	if closer, ok := any(value).(io.Closer); ok {
		scope.OnClose(closer.Close)
	} else if closer, ok := any(&value).(io.Closer); ok {
		scope.OnClose(closer.Close)
	}
}

// ResourceScope collects cleanup callbacks in reverse registration order.
// Register before cleanup begins; Close can then be called repeatedly or concurrently.
type ResourceScope struct {
	releases []func() error
	once     sync.Once
	err      error
}

// OnClose registers release for [ResourceScope.Close]; nil callbacks are ignored.
func (scope *ResourceScope) OnClose(release func() error) {
	if release != nil {
		scope.releases = append(scope.releases, release)
	}
}

// Close runs cleanup once and returns joined errors. Repeated calls return that same result.
func (scope *ResourceScope) Close() error {
	if scope == nil {
		return nil
	}
	scope.once.Do(func() {
		releases := scope.releases
		scope.releases = nil
		var errs []error
		for _, release := range slices.Backward(releases) {
			if err := release(); err != nil {
				errs = append(errs, err)
			}
		}
		scope.err = errors.Join(errs...)
	})
	return scope.err
}

// CloneRequirement copies configuration without resolving a runtime value.
// It uses [RequirementCloner] when available, otherwise lossless serialization.
func CloneRequirement[V, R any](ctx context.Context, source Codec[V, R], requirement R) (R, error) {
	var zero R
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if clone, ok := source.(RequirementCloner[R]); ok {
		return clone.CloneRequirement(ctx, requirement)
	}
	plain, err := source.FormatRequirement(ctx, requirement)
	if err != nil {
		return zero, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return source.ParseRequirement(ctx, UsageDefault, plain...)
}

// DisplayRequirement returns help-safe configuration text without acquiring resources.
func DisplayRequirement[V, R any](ctx context.Context, source Codec[V, R], requirement R) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return source.DisplayRequirement(ctx, requirement)
}
