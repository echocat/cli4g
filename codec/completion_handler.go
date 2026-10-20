package codec

import (
	"context"
	"fmt"
	"reflect"
	"slices"
)

// WithCompletionHandler attaches a context-aware value handler to an existing
// codec without requiring a new codec type or ToRaw method. For example:
//
//	codec.WithCompletionHandler(codec.NewString(), completeNamespaces)
//
// Parsing, validation, defaults and resource resolution retain the source codec's
// behavior. Enum metadata, collection accumulation, bool flags and type binding
// remain available. The handler runs only for value completion, never while
// creating the definition, rendering help or parsing an ordinary invocation.
//
// Handled=true is authoritative, including an empty answer. Handled=false delegates
// to the source codec's completer and automatic enum/bool/path suggestions. Return
// literal, unquoted tokens; the framework filters the prefix and the shell quotes
// inserted values. The handler must honor cancellation and be concurrent-safe.
// A nil handler returns source unchanged. A nil source panics, like NewSlice.
func WithCompletionHandler[V, R any](source Codec[V, R], handler CompletionFunc) Codec[V, R] {
	if isNil(source) {
		panic("completion handler requires a codec")
	}
	if handler == nil {
		return source
	}
	result := completingCodec[V, R]{Codec: source, handler: handler}
	if enumeration, ok := source.(Enum[R]); ok {
		return completingEnum[V, R]{completingCodec: result, enumeration: enumeration}
	}
	return result
}

type completingCodec[V, R any] struct {
	Codec[V, R]
	handler CompletionFunc
}

func (this completingCodec[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	return Raw(ctx, this)
}

func (this completingCodec[V, R]) Complete(ctx context.Context, input CompletionInput) (CompletionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CompletionResult{}, err
	}
	result, err := this.handler(ctx, input)
	if err != nil {
		return CompletionResult{}, SanitizeError(err, "completion handler failed")
	}
	if err := ctx.Err(); err != nil {
		return CompletionResult{}, err
	}
	if result.Handled {
		result.Candidates = slices.Clone(result.Candidates)
		return result, nil
	}
	return Complete(ctx, this.Codec, input)
}

func (this completingCodec[V, R]) CloneRequirement(ctx context.Context, value R) (R, error) {
	return CloneRequirement(ctx, this.Codec, value)
}

func (this completingCodec[V, R]) RequirementType() reflect.Type {
	typ := reflect.TypeFor[R]()
	if typ.Kind() == reflect.Interface {
		if source, ok := this.Codec.(interface{ RequirementType() reflect.Type }); ok {
			return source.RequirementType()
		}
	}
	return typ
}

func (this completingCodec[V, R]) EnumChoices(ctx context.Context) ([]EnumChoice, error) {
	return DisplayEnum(ctx, this.Codec)
}

func (this completingCodec[V, R]) IsCumulative() bool { return IsCumulative(this.Codec) }

func (this completingCodec[V, R]) ParseRequirementInto(ctx context.Context, usage Usage, previous R, args ...string) (R, error) {
	if source, ok := this.Codec.(Cumulative[R]); ok {
		return source.ParseRequirementInto(ctx, usage, previous, args...)
	}
	return previous, fmt.Errorf("%w: codec has no cumulative requirement parser", ErrInvalid)
}

func (this completingCodec[V, R]) IsBoolFlag() bool {
	if reflect.TypeFor[V]().Kind() == reflect.Bool {
		return true
	}
	source, ok := this.Codec.(interface{ IsBoolFlag() bool })
	return ok && source.IsBoolFlag()
}

func (this completingCodec[V, R]) BindType(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	if source, ok := this.Codec.(TypeBinder); ok {
		bound, matched, err := source.BindType(ctx, typ)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil || !matched {
			return nil, matched, err
		}
		if isNil(bound) {
			return nil, false, fmt.Errorf("%w: type binder returned a nil codec", ErrInvalid)
		}
		result, err := Snapshot(ctx, WithCompletionHandler(bound, this.handler))
		return result, err == nil, err
	}
	matched, err := this.Codec.CanHandleType(ctx, typ)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil || !matched {
		return nil, matched, err
	}
	result, err := Snapshot(ctx, WithCompletionHandler(this.Codec, this.handler))
	return result, err == nil, err
}

type completingEnum[V, R any] struct {
	completingCodec[V, R]
	enumeration Enum[R]
}

func (this completingEnum[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	return Raw(ctx, this)
}

func (this completingEnum[V, R]) EnumVariants(ctx context.Context) ([]EnumVariant[R], error) {
	return this.enumeration.EnumVariants(ctx)
}
