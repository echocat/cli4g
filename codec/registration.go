package codec

import "slices"

// Registration binds a typed codec to zero or more names. No names mean that
// the codec is considered only for options fields without a type tag.
type Registration struct {
	Codec any
	Names []string
}

func NewRegistration(codec any, names ...string) Registration {
	return Registration{codec, slices.Clone(names)}
}
