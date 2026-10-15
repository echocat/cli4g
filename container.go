package cli

import (
	"iter"
	"reflect"
)

// Container exposes metadata common to applications and commands. Options and
// handler types remain specific to their respective typed definitions.
// Applications and commands implement it; callers cannot implement it externally.
type Container interface {
	Name() string
	Names() []string
	Description() string
	Commands() iter.Seq[CommandLike]
	Command(string) (CommandLike, bool)
	Flags() iter.Seq[*Flag[any, any]]
	Flag(string) (*Flag[any, any], bool)
	ShortFlag(rune) (*Flag[any, any], bool)
	Arguments() iter.Seq[*Argument[any, any]]
	Argument(int) (*Argument[any, any], bool)
	OptionsType() reflect.Type
	base() *containerBase
}
