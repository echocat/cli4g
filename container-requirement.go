package cli

import (
	"context"
	"reflect"

	"github.com/echocat/cli4g/codec"
)

type ContainerRequirementSetters[O any] interface {
	SetCommands([]CommandRequirementLike)
	SetCodecs([]codec.Registration)
	SetFlags([]FlagRequirementLike)
	SetArguments([]ArgumentRequirementLike)
	OptionsType() reflect.Type
	SetCreateOptionsHandler(func(context.Context) (O, error))
	SetValidateHandler(func(context.Context, O) error)
	SetInitializeHandler(func(context.Context, O) error)
	SetHandler(func(context.Context, O) error)
	SetCloseHandler(func(O) error)
}

type ContainerRequirement[O any, R any] interface {
	ContainerRequirementSetters[O]
	WithCommand(...CommandRequirementLike) R
	WithCommandP(...CommandRequirementLike) R
	WithCodec(any, ...string) R
	WithFlag(...FlagRequirementLike) R
	WithArgument(...ArgumentRequirementLike) R
	WithCreateOptionsHandler(func(context.Context) (O, error)) R
	WithValidateHandler(func(context.Context, O) error) R
	WithInitializeHandler(func(context.Context, O) error) R
	WithHandler(func(context.Context, O) error) R
	WithCloseHandler(func(O) error) R
}
