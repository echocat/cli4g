package cli

import (
	"github.com/echocat/cli4g/codec"
)

type ContainerRequirementSetters interface {
	AddCommand(...CommandRequirementLike)
	SetCommands([]CommandRequirementLike)

	AddCodec(any, ...string)
	SetCodecs([]codec.Registration)

	AddFlag(...FlagRequirementLike)
	SetFlags([]FlagRequirementLike)

	AddArgument(...ArgumentRequirementLike)
	SetArguments([]ArgumentRequirementLike)
}

type ContainerRequirement[R any] interface {
	ContainerRequirementSetters
	WithCommand(...CommandRequirementLike) R
	WithCodec(any, ...string) R
	WithFlag(...FlagRequirementLike) R
	WithArgument(...ArgumentRequirementLike) R
}
