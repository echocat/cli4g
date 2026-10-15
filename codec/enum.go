package codec

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Enum is an optional codec contract describing every valid requirement, in
// declaration order. The set must be nonempty and independent of context values.
// Return resource-free, independent copies. Metadata methods must not
// acquire resources or resolve runtime values. Contexts provide cancellation.
// A codec implementing Enum must enforce the advertised set itself.
type Enum[R any] interface {
	EnumVariants(context.Context) ([]EnumVariant[R], error)
}

// EnumVariant describes one canonical requirement and its optional help text.
type EnumVariant[R any] struct {
	Value       R
	Description string
}

// EnumChoice is a display-safe variant, possibly containing multiple tokens.
// Values come from DisplayRequirement, never from runtime value formatting.
type EnumChoice struct {
	Values      []string
	Description string
}

// EnumChoices exposes display-only choices, including element choices for
// collections. Nil means no choices. Unlike Enum, it does not assert that the
// codec's entire requirement space is finite. Raw and collection adapters use
// this projection to preserve choices without pretending collections are enums.
type EnumChoices interface {
	EnumChoices(context.Context) ([]EnumChoice, error)
}

// DisplayEnum returns independent, display-safe choices without Resolve calls.
// It supports both typed Enum implementations and collection projections.
func DisplayEnum[V, R any](ctx context.Context, source Codec[V, R]) ([]EnumChoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if enumeration, ok := source.(Enum[R]); ok {
		variants, err := enumeration.EnumVariants(ctx)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(variants) == 0 {
			return nil, fmt.Errorf("%w: enum requires at least one variant", ErrInvalid)
		}
		result := make([]EnumChoice, len(variants))
		seen := make(map[string]bool, len(variants))
		for i, variant := range variants {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if err := ValidateRequirement(variant.Value); err != nil {
				return nil, err
			}
			plain, err := source.FormatRequirement(ctx, variant.Value)
			if err != nil {
				return nil, err
			}
			key := enumKey(plain)
			if seen[key] {
				return nil, fmt.Errorf("%w: duplicate enum variant", ErrInvalid)
			}
			seen[key] = true
			values, err := DisplayRequirement(ctx, source, variant.Value)
			if err != nil {
				return nil, err
			}
			result[i] = EnumChoice{Values: slices.Clone(values), Description: variant.Description}
		}
		return result, ctx.Err()
	}
	if projection, ok := source.(EnumChoices); ok {
		choices, err := projection.EnumChoices(ctx)
		if err != nil {
			return nil, err
		}
		result := slices.Clone(choices)
		for i := range result {
			result[i].Values = slices.Clone(result[i].Values)
		}
		return result, ctx.Err()
	}
	return nil, ctx.Err()
}

// NewEnum restricts a codec to the supplied resource-free requirements. Parsing
// retains the source codec's normalization rules. Unknown requirements fail
// before Resolve. Use NewSlice/NewMap around an enum for repeated elements;
// an enum itself selects one variant and is not cumulative.
func NewEnum[V, R any](source Codec[V, R], values ...R) (Codec[V, R], error) {
	return NewEnumVariants(source, enumVariants(values)...)
}

func enumVariants[R any](values []R) []EnumVariant[R] {
	variants := make([]EnumVariant[R], len(values))
	for i, value := range values {
		variants[i].Value = value
	}
	return variants
}

// NewEnumVariants is NewEnum with per-variant descriptions. It snapshots all
// requirements, preserving declaration order and rejecting duplicate encodings.
// Constructors never resolve values or call Set/UnmarshalText.
func NewEnumVariants[V, R any](source Codec[V, R], variants ...EnumVariant[R]) (Codec[V, R], error) {
	if isNil(source) || len(variants) == 0 {
		return nil, fmt.Errorf("%w: enum requires a codec and at least one variant", ErrInvalid)
	}
	ctx := context.Background()
	if _, err := Snapshot(ctx, source); err != nil {
		return nil, err
	}
	result := enumCodec[V, R]{source: source, variants: make([]EnumVariant[R], len(variants)), allowed: make(map[string]bool, len(variants))}
	for i, variant := range variants {
		if err := ValidateRequirement(variant.Value); err != nil {
			return nil, err
		}
		value, err := CloneRequirement(ctx, source, variant.Value)
		if err != nil {
			return nil, err
		}
		if err := ValidateRequirement(value); err != nil {
			return nil, err
		}
		plain, err := source.FormatRequirement(ctx, value)
		if err != nil {
			return nil, err
		}
		key := enumKey(plain)
		if result.allowed[key] {
			return nil, fmt.Errorf("%w: duplicate enum variant", ErrInvalid)
		}
		result.allowed[key] = true
		result.variants[i] = EnumVariant[R]{Value: value, Description: variant.Description}
	}
	choices, err := DisplayEnum(ctx, result)
	if err != nil {
		return nil, err
	}
	labels := make([]string, len(choices))
	for i, choice := range choices {
		labels[i] = enumLabel(choice.Values)
	}
	result.expected = strings.Join(labels, ", ")
	return result, nil
}

type enumCodec[V, R any] struct {
	source   Codec[V, R]
	variants []EnumVariant[R]
	allowed  map[string]bool
	expected string
}

func (this enumCodec[V, R]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }
func (this enumCodec[V, R]) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return this.source.CanHandleType(ctx, typ)
}
func (this enumCodec[V, R]) ExpectMinMax(ctx context.Context, usage Usage) (int, int) {
	return this.source.ExpectMinMax(ctx, usage)
}
func (this enumCodec[V, R]) IsBoolFlag() bool {
	if reflect.TypeFor[V]().Kind() == reflect.Bool {
		return true
	}
	trait, ok := this.source.(interface{ IsBoolFlag() bool })
	return ok && trait.IsBoolFlag()
}
func (this enumCodec[V, R]) EnumVariants(ctx context.Context) ([]EnumVariant[R], error) {
	result := make([]EnumVariant[R], len(this.variants))
	for i, variant := range this.variants {
		value, err := CloneRequirement(ctx, this.source, variant.Value)
		if err != nil {
			return nil, err
		}
		result[i] = EnumVariant[R]{Value: value, Description: variant.Description}
	}
	return result, ctx.Err()
}
func (this enumCodec[V, R]) check(ctx context.Context, value R) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	plain, err := this.source.FormatRequirement(ctx, value)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if this.allowed[enumKey(plain)] {
		return nil
	}
	display, err := DisplayRequirement(ctx, this.source, value)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: invalid enum value %s; expected one of: %s", ErrIllegalArgument, enumLabel(display), this.expected)
}
func (this enumCodec[V, R]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (R, error) {
	value, err := this.source.ParseRequirement(ctx, usage, args...)
	if err != nil {
		if IsValidationError(err) {
			return value, fmt.Errorf("%w; expected one of: %s", err, this.expected)
		}
		return value, err
	}
	if err := this.check(ctx, value); err != nil {
		return value, err
	}
	return value, nil
}
func (this enumCodec[V, R]) FormatRequirement(ctx context.Context, value R) ([]string, error) {
	if err := this.check(ctx, value); err != nil {
		return nil, err
	}
	return this.source.FormatRequirement(ctx, value)
}
func (this enumCodec[V, R]) DisplayRequirement(ctx context.Context, value R) ([]string, error) {
	if err := this.check(ctx, value); err != nil {
		return nil, err
	}
	return DisplayRequirement(ctx, this.source, value)
}
func (this enumCodec[V, R]) CloneRequirement(ctx context.Context, value R) (R, error) {
	if err := this.check(ctx, value); err != nil {
		var zero R
		return zero, err
	}
	return CloneRequirement(ctx, this.source, value)
}
func (this enumCodec[V, R]) Resolve(ctx context.Context, scope Scope, value R) (V, error) {
	if err := this.check(ctx, value); err != nil {
		var zero V
		return zero, err
	}
	return this.source.Resolve(ctx, scope, value)
}

// Length-prefixed tokens preserve boundaries, nil/empty distinctions and opaque
// bytes without the UTF-8 replacement that JSON string encoding would perform.
func enumKey(values []string) string {
	if values == nil {
		return "nil"
	}
	var result strings.Builder
	result.WriteByte('[')
	for _, value := range values {
		result.WriteString(strconv.Itoa(len(value)))
		result.WriteByte(':')
		result.WriteString(value)
	}
	return result.String()
}

func enumLabel(values []string) string {
	if len(values) == 1 {
		return strconv.Quote(values[0])
	}
	return fmt.Sprintf("%q", values)
}
