package cli

import (
	"os"

	"github.com/echocat/cli4g/codec"
)

// WithFileFlag adds a read-only file flag. Defaults use [codec.FileRequirement];
// handles are invocation-owned. Select write policy with [FlagRequirement.WithCodec].
func (this *ApplicationRequirement[O]) WithFileFlag(name string, configure ...func(*FlagRequirement[*os.File, codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFile(), name, configure...))
}

// WithFileSliceFlag adds a repeatable read-only file flag; see [codec.NewSlice].
func (this *ApplicationRequirement[O]) WithFileSliceFlag(name string, configure ...func(*FlagRequirement[[]*os.File, []codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewSlice(codec.NewFile()), name, configure...))
}

// WithFileArgument adds a read-only file argument; defaults use [codec.FileRequirement].
func (this *ApplicationRequirement[O]) WithFileArgument(name string, configure ...func(*ArgumentRequirement[*os.File, codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFile(), name, configure...))
}

// WithFileSliceArgument adds variadic read-only files; it must be last.
func (this *ApplicationRequirement[O]) WithFileSliceArgument(name string, configure ...func(*ArgumentRequirement[[]*os.File, []codec.FileRequirement])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewSlice(codec.NewFile()), name, configure...))
}
