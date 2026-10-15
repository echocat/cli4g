package codec

// Value is a flag.Value-style Set/String value. Set validates individual inputs;
// String is display text, not a lossless serialization contract.
type Value interface {
	String() string
	Set(string) error
}

// NewValue adapts *T's Set/String methods. The zero value must be ready for Set.
// Requirements and defaults are input tokens ([]string), not T. Set runs during
// Resolve; mark user errors with [NewValidationError]. See [ValueConfig] for traits.
func NewValue[T any, TP interface {
	*T
	Value
}](configs ...ValueConfig[T]) (Codec[T, []string], error) {
	return NewFlagValue[T, TP](configs...)
}

// NewValueP adapts fresh targets from a concurrent-safe factory, called during Resolve.
// Nil factories fail configuration; nil targets fail with [ErrInvalid].
// Requirements are []string; see [NewValue] and [ValueConfig].
func NewValueP[T Value](factory func() T, configs ...ValueConfig[T]) (Codec[T, []string], error) {
	if factory == nil {
		return NewPointerFlagValue[T](nil, configs...)
	}
	return NewPointerFlagValue(func() (T, error) { return factory(), nil }, configs...)
}
