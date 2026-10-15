package codec

import "reflect"

// Shared immutable metadata for a primitive numeric type.
type numericConfig struct {
	typ  reflect.Type
	bits uint8
}
