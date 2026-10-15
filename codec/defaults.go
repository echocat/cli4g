package codec

import "context"

// Defaults are appended after user-registered codecs while Create runs.
func Defaults(ctx context.Context) []Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return []Codec[any, any]{
		NewBool().ToRaw(ctx),
		NewString().ToRaw(ctx),
		NewDuration().ToRaw(ctx),
		NewDateTime().ToRaw(ctx),
		NewInt().ToRaw(ctx),
		NewInt8().ToRaw(ctx),
		NewInt16().ToRaw(ctx),
		NewInt32().ToRaw(ctx),
		NewInt64().ToRaw(ctx),
		NewUint().ToRaw(ctx),
		NewUint8().ToRaw(ctx),
		NewUint16().ToRaw(ctx),
		NewUint32().ToRaw(ctx),
		NewUint64().ToRaw(ctx),
		NewUintptr().ToRaw(ctx),
		NewFloat32().ToRaw(ctx),
		NewFloat64().ToRaw(ctx),
		NewComplex64().ToRaw(ctx),
		NewComplex128().ToRaw(ctx),

		NewBoolSlice().ToRaw(ctx),
		NewStringSlice().ToRaw(ctx),
		NewDurationSlice().ToRaw(ctx),
		NewDateTimeSlice().ToRaw(ctx),
		NewIntSlice().ToRaw(ctx),
		NewInt8Slice().ToRaw(ctx),
		NewInt16Slice().ToRaw(ctx),
		NewInt32Slice().ToRaw(ctx),
		NewInt64Slice().ToRaw(ctx),
		NewUintSlice().ToRaw(ctx),
		NewUint8Slice().ToRaw(ctx),
		NewUint16Slice().ToRaw(ctx),
		NewUint32Slice().ToRaw(ctx),
		NewUint64Slice().ToRaw(ctx),
		NewUintptrSlice().ToRaw(ctx),
		NewFloat32Slice().ToRaw(ctx),
		NewFloat64Slice().ToRaw(ctx),
		NewComplex64Slice().ToRaw(ctx),
		NewComplex128Slice().ToRaw(ctx),
	}
}
