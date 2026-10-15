package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// invocationFrame belongs to one application or selected command. Options are
// read from their canonical instance. Decoded values are stored separately, so
// replacing an options field does not replace its decoded value. Reference
// values can still share their underlying data with the options.
type invocationFrame struct {
	container Container
	value     reflect.Value
	flags     []flagEntry[any]
	arguments []argumentEntry[any]
	resources *codec.ResourceScope
}

// The frame owns the addressable options instance for its entire lifetime,
// independently of the temporary parseScope that populates its values.
func (this *invocationFrame) currentOptions() any {
	if !this.value.IsValid() {
		return nil
	}
	return this.value.Interface()
}

func (this *invocationFrame) lifecycleOptions() any {
	if !this.value.IsValid() {
		return nil
	}
	if this.value.Kind() != reflect.Pointer {
		return this.value.Addr().Interface()
	}
	return this.value.Interface()
}

func (this *invocationFrame) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	validator, hasValidator := this.lifecycleOptions().(Validatable)
	handler := this.container.base().validateHandler
	if !hasValidator && handler == nil {
		return nil
	}
	ctx = withContainerContext(ctx, this.container)
	if hasValidator {
		if err := validator.Validate(ctx); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if handler != nil {
		if err := handler(ctx, this.currentOptions()); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (this *invocationFrame) initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	initializer, hasInitializer := this.lifecycleOptions().(Initializable)
	handler := this.container.base().initializeHandler
	if hasInitializer || handler != nil {
		ctx = withContainerContext(ctx, this.container)
	}
	if hasInitializer {
		if err := initializer.Initialize(ctx); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if handler != nil {
		if err := handler(ctx, this.currentOptions()); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Owned options are closed even after partial decoding or initialization.
func (this *invocationFrame) close() error {
	var errs []error
	if this.value.IsValid() && this.container.base().closeHandler != nil {
		if err := this.container.base().close(this.currentOptions()); err != nil {
			errs = append(errs, err)
		}
	}
	if closer, ok := this.lifecycleOptions().(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if this.resources != nil {
		if err := this.resources.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Direct calls operate on the caller's pointer options. Struct options follow
// Go's value semantics: create an addressable copy for pointer-receiver methods.
func directOptionsFrame(container Container, options any) invocationFrame {
	value := reflect.ValueOf(options)
	if value.IsValid() && value.Kind() != reflect.Pointer {
		owned := reflect.New(value.Type()).Elem()
		owned.Set(value)
		value = owned
	}
	return invocationFrame{container: container, value: value}
}

func (this *invocationFrame) OnClose(release func() error) {
	if release == nil {
		return
	}
	if this.resources == nil {
		this.resources = new(codec.ResourceScope)
	}
	this.resources.OnClose(release)
}

func (this *invocationFrame) resolve(ctx context.Context) error {
	for i := range this.flags {
		entry := &this.flags[i]
		value, err := entry.flag.codec.Resolve(ctx, this, entry.requirement)
		if err != nil {
			return fmt.Errorf("flag --%s: %w", entry.flag.Name(), err)
		}
		if err := bindResolved(this.value, entry.flag.binding, entry.flag.valueType, value); err != nil {
			return fmt.Errorf("flag --%s: %w", entry.flag.Name(), err)
		}
		entry.value = value
		entry.requirement = nil
		entry.resolved = true
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	for i := range this.arguments {
		entry := &this.arguments[i]
		value, err := entry.argument.codec.Resolve(ctx, this, entry.requirement)
		if err != nil {
			return fmt.Errorf("argument %s: %w", entry.argument.Name(), err)
		}
		if err := bindResolved(this.value, entry.argument.binding, entry.argument.valueType, value); err != nil {
			return fmt.Errorf("argument %s: %w", entry.argument.Name(), err)
		}
		entry.value = value
		entry.requirement = nil
		entry.resolved = true
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func bindResolved(options reflect.Value, binding *fieldBinding, typ reflect.Type, value any) error {
	if err := checkValueType(value, typ); err != nil {
		return err
	}
	if binding != nil {
		return binding.set(options, value)
	}
	return nil
}
func checkValueType(value any, typ reflect.Type) error {
	if value == nil {
		switch typ.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
			return nil
		}
	} else if reflect.TypeOf(value).AssignableTo(typ) {
		return nil
	}
	return fmt.Errorf("codec returned %T, expected %s", value, typ)
}

func (this invocationFrame) eachFlag() iter.Seq2[*Flag[any, any], any] {
	return func(yield func(*Flag[any, any], any) bool) {
		for _, e := range this.flags {
			if !e.resolved {
				continue
			}
			if !yield(e.flag, e.value) {
				return
			}
		}
	}
}

func (this invocationFrame) flag(name string) (*Flag[any, any], any) {
	for _, e := range this.flags {
		if e.resolved && slices.Contains(e.flag.names, name) {
			return e.flag, e.value
		}
	}
	return nil, nil
}

func (this invocationFrame) shortFlag(short rune) (*Flag[any, any], any) {
	for _, e := range this.flags {
		if e.resolved && slices.Contains(e.flag.shorts, short) {
			return e.flag, e.value
		}
	}
	return nil, nil
}

func (this invocationFrame) eachArgument() iter.Seq2[*Argument[any, any], any] {
	return func(yield func(*Argument[any, any], any) bool) {
		for _, e := range this.arguments {
			if !e.resolved {
				continue
			}
			if !yield(e.argument, e.value) {
				return
			}
		}
	}
}

func (this invocationFrame) argument(i int) (*Argument[any, any], any) {
	if i < 0 {
		return nil, nil
	}
	// Dense definitions keep the direct-access fast path. Stored entries are
	// otherwise compact, so omitted arguments must not shift later indices.
	if i < len(this.arguments) {
		e := this.arguments[i]
		if e.resolved && e.argument.index == i {
			return e.argument, e.value
		}
	}
	for _, e := range this.arguments {
		if e.resolved && e.argument.index == i {
			return e.argument, e.value
		}
		if e.argument.index > i {
			break
		}
	}
	return nil, nil
}
