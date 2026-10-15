package cli

import (
	"encoding"
	"os"
	"time"

	"github.com/echocat/cli4g/codec"
)

// WithBoolMapFlag adds a repeatable map[string]bool flag. Map helpers accept one
// key=value pair per occurrence and reject duplicate keys. Bind via WithField.
func (this *ApplicationRequirement[O]) WithBoolMapFlag(name string, configure ...func(*FlagRequirement[map[string]bool, map[string]bool])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewBoolMap(), name, configure...))
}

// WithStringMapFlag adds a repeatable map[string]string flag.
func (this *ApplicationRequirement[O]) WithStringMapFlag(name string, configure ...func(*FlagRequirement[map[string]string, map[string]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewStringMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDurationMapFlag(name string, configure ...func(*FlagRequirement[map[string]time.Duration, map[string]time.Duration])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDurationMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDateMapFlag(name string, configure ...func(*FlagRequirement[map[string]time.Time, map[string]time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDateTimeMapFlag(name string, configure ...func(*FlagRequirement[map[string]time.Time, map[string]time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateTimeMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntMapFlag(name string, configure ...func(*FlagRequirement[map[string]int, map[string]int])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewIntMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8MapFlag(name string, configure ...func(*FlagRequirement[map[string]int8, map[string]int8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt8Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16MapFlag(name string, configure ...func(*FlagRequirement[map[string]int16, map[string]int16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt16Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32MapFlag(name string, configure ...func(*FlagRequirement[map[string]int32, map[string]int32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64MapFlag(name string, configure ...func(*FlagRequirement[map[string]int64, map[string]int64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintMapFlag(name string, configure ...func(*FlagRequirement[map[string]uint, map[string]uint])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8MapFlag(name string, configure ...func(*FlagRequirement[map[string]uint8, map[string]uint8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint8Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16MapFlag(name string, configure ...func(*FlagRequirement[map[string]uint16, map[string]uint16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint16Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32MapFlag(name string, configure ...func(*FlagRequirement[map[string]uint32, map[string]uint32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64MapFlag(name string, configure ...func(*FlagRequirement[map[string]uint64, map[string]uint64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrMapFlag(name string, configure ...func(*FlagRequirement[map[string]uintptr, map[string]uintptr])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintptrMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32MapFlag(name string, configure ...func(*FlagRequirement[map[string]float32, map[string]float32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64MapFlag(name string, configure ...func(*FlagRequirement[map[string]float64, map[string]float64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64MapFlag(name string, configure ...func(*FlagRequirement[map[string]complex64, map[string]complex64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128MapFlag(name string, configure ...func(*FlagRequirement[map[string]complex128, map[string]complex128])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex128Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerMapFlag[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*FlagRequirement[map[string]T, map[string]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMarshallerMap[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPMapFlag[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*FlagRequirement[map[string]T, map[string]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(marshallerPointerMapFactory(factory), name, configure...))
}

func (this *ApplicationRequirement[O]) WithValueMapFlag[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*FlagRequirement[map[string]T, map[string][]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[map[string]T, map[string][]string], error) {
		return codec.NewValueMap[T, TP]()
	}, name, configure...))
}

func (this *ApplicationRequirement[O]) WithValuePMapFlag[T codec.Value](name string, factory func() T, configure ...func(*FlagRequirement[map[string]T, map[string][]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[map[string]T, map[string][]string], error) {
		return codec.NewValuePMap(factory)
	}, name, configure...))
}

func (this *ApplicationRequirement[O]) WithFileMapFlag(name string, configure ...func(*FlagRequirement[map[string]*os.File, map[string]codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMap[string](codec.NewFile()), name, configure...))
}

// WithBoolMapArgument adds a final variadic map[string]bool argument. Map
// arguments accept key=value pairs and reject duplicate keys.
func (this *ApplicationRequirement[O]) WithBoolMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]bool, map[string]bool])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewBoolMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithStringMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]string, map[string]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewStringMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDurationMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]time.Duration, map[string]time.Duration])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDurationMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDateMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]time.Time, map[string]time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithDateTimeMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]time.Time, map[string]time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateTimeMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]int, map[string]int])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewIntMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]int8, map[string]int8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt8Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]int16, map[string]int16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt16Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]int32, map[string]int32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]int64, map[string]int64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uint, map[string]uint])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uint8, map[string]uint8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint8Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uint16, map[string]uint16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint16Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uint32, map[string]uint32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uint64, map[string]uint64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]uintptr, map[string]uintptr])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintptrMap(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]float32, map[string]float32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat32Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]float64, map[string]float64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]complex64, map[string]complex64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex64Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128MapArgument(name string, configure ...func(*ArgumentRequirement[map[string]complex128, map[string]complex128])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex128Map(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerMapArgument[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*ArgumentRequirement[map[string]T, map[string]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMarshallerMap[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPMapArgument[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*ArgumentRequirement[map[string]T, map[string]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(marshallerPointerMapFactory(factory), name, configure...))
}

func (this *ApplicationRequirement[O]) WithValueMapArgument[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*ArgumentRequirement[map[string]T, map[string][]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[map[string]T, map[string][]string], error) {
		return codec.NewValueMap[T, TP]()
	}, name, configure...))
}

func (this *ApplicationRequirement[O]) WithValuePMapArgument[T codec.Value](name string, factory func() T, configure ...func(*ArgumentRequirement[map[string]T, map[string][]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[map[string]T, map[string][]string], error) {
		return codec.NewValuePMap(factory)
	}, name, configure...))
}

func (this *ApplicationRequirement[O]) WithFileMapArgument(name string, configure ...func(*ArgumentRequirement[map[string]*os.File, map[string]codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMap[string](codec.NewFile()), name, configure...))
}
