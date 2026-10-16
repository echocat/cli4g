package codec

import "context"

// Defaults supplies fallback codecs after user registrations during Create.
// Its initial value returns one CombinedDefaults resolver. Replacing this function
// affects subsequent Create calls, not compiled definitions. Configure it before
// Create, without concurrent modification. Nil disables the standard fallback.
var Defaults = func(ctx context.Context) []Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return []Codec[any, any]{NewCombinedDefaults()}
}
