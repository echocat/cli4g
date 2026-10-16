package cli

import (
	"context"
	"reflect"
	"slices"
)

// ApplicationFromContext returns the current root options, without cloning or
// resolving anything. Missing invocation/options or a wrong type returns false.
// Reference fields share data with the invocation; struct options are copied.
func ApplicationFromContext[O any](ctx context.Context) (O, bool) {
	var zero O
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return zero, false
	}
	value, found := invocation.optionsFor(invocation.Root())
	if !found {
		return zero, false
	}
	return contextValueAs[O](value)
}

// CommandFromContext returns the current command's options when path is empty.
// An explicit path starts at the application and must identify a command selected
// in this invocation. It follows the selected stack, including profile-specific
// variants and ancestors. Aliases are accepted; no options are created by this getter.
func CommandFromContext[O any](ctx context.Context, path ...string) (O, bool) {
	var zero O
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return zero, false
	}
	var container Container
	if len(path) == 0 {
		current, found := ContainerFromContext(ctx)
		if !found {
			return zero, false
		}
		if _, command := current.(CommandLike); !command {
			return zero, false
		}
		container = current
	} else {
		stack := invocation.Stack()
		if len(path) > len(stack) {
			return zero, false
		}
		for i, name := range path {
			if !slices.Contains(stack[i].Command.Names(), name) {
				return zero, false
			}
		}
		container = stack[len(path)-1].Command
	}
	value, found := invocation.optionsFor(container)
	if !found {
		return zero, false
	}
	return contextValueAs[O](value)
}

// FlagFromContext returns the existing decoded value, including declared defaults
// and environment inputs. Factory-only fields are available through options.
// A present typed nil, empty string, false or zero remains found=true.
func FlagFromContext[V any](ctx context.Context, name string) (V, bool) {
	var zero V
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return zero, false
	}
	definition, value := invocation.Flag(name)
	if definition == nil {
		return zero, false
	}
	return contextValueAs[V](value)
}

func contextValueAs[T any](value any) (T, bool) {
	typed, found := value.(T)
	if found {
		return typed, true
	}
	if value == nil && reflect.TypeFor[T]().Kind() == reflect.Interface {
		return typed, true
	}
	return typed, false
}
