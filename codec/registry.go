package codec

import (
	"context"
	"fmt"
	"reflect"
	"slices"
)

// Registry is used while compiling a CLI definition. Compiled flags and
// arguments retain only the selected codec, not a reference to the registry.
type Registry struct {
	entries []entry
}

type entry struct {
	codec Codec[any, any]
	names []string
}

func NewRegistry(ctx context.Context, registrations []Registration) (Registry, error) {
	if err := ctx.Err(); err != nil {
		return Registry{}, err
	}
	var registry Registry
	for _, registration := range registrations {
		if slices.Contains(registration.Names, "") {
			return Registry{}, fmt.Errorf("empty codec name")
		}
		snapshot, err := Snapshot(ctx, registration.Codec)
		if err != nil {
			return Registry{}, err
		}
		registry.entries = append(registry.entries, entry{codec: snapshot, names: slices.Clone(registration.Names)})
	}
	return registry, nil
}

// AddDefaults adds builtins after the explicitly registered codecs.
func (this *Registry) AddDefaults(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if Defaults == nil {
		return nil
	}
	defaults := Defaults(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	this.entries = slices.Grow(this.entries, len(defaults))
	for _, builtin := range defaults {
		snapshot, err := Snapshot(ctx, builtin)
		if err != nil {
			return err
		}
		this.entries = append(this.entries, entry{codec: snapshot})
	}
	return ctx.Err()
}

func (this Registry) Find(ctx context.Context, typ reflect.Type, name string) (Codec[any, any], bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	for _, candidate := range this.entries {
		if len(candidate.names) == 0 && name != "" || len(candidate.names) != 0 && !slices.Contains(candidate.names, name) {
			continue
		}
		if binder, ok := candidate.codec.(TypeBinder); ok {
			bound, matched, err := binder.BindType(ctx, typ)
			if err == nil {
				err = ctx.Err()
			}
			if err != nil {
				return nil, false, err
			}
			if matched {
				snapshot, err := Snapshot(ctx, bound)
				return snapshot, err == nil, err
			}
			continue
		}
		matches, err := candidate.codec.CanHandleType(ctx, typ)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			return nil, false, err
		}
		if matches {
			return candidate.codec, true, nil
		}
	}
	return nil, false, nil
}
