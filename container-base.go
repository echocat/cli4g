package cli

import (
	"context"
	"iter"
	"reflect"
	"slices"
)

// containerBase is the compiled heterogeneous base representation of a container.
// Typed builders bind their own options factory and handler adapters during Create.
type containerBase struct {
	containerSchema
	commands             []CommandLike
	commandsByName       map[string]CommandLike
	plan                 *parsePlan
	unprofiled           selectedContainer
	newOptions           func() reflect.Value
	createOptionsHandler func(context.Context) (reflect.Value, error)

	validateHandler   func(context.Context, any) error
	initializeHandler func(context.Context, any) error
	handler           func(context.Context, any) error
	closeHandler      func(any) error
}

func (this *containerBase) base() *containerBase {
	return this
}

func (this *containerBase) prepareContext(ctx context.Context) context.Context {
	var definition *definitionLifecycle
	if this.plan != nil {
		definition = this.plan.definition
	}
	return definition.prepareContext(ctx)
}

// Direct container operations dispatch new work only while their owning
// application is open. Cleanup and immutable metadata remain available.
func (this *containerBase) requireOpen() error {
	if this.plan != nil && this.plan.definition != nil && this.plan.definition.closed.Load() {
		return ErrApplicationClosed
	}
	return nil
}

func (this *containerBase) OptionsType() reflect.Type {
	return this.optionsType
}

func (this *containerBase) Commands() iter.Seq[CommandLike] {
	return slices.Values(this.commands)
}

// CommandOfProfile finds a command explicitly assigned to profile, by name or
// alias. It does not fall back to commands without profile assignments.
func (this *containerBase) CommandOfProfile(name, profile string) (CommandLike, bool) {
	for _, command := range this.commands {
		if slices.Contains(command.Profiles(), profile) && slices.Contains(command.Names(), name) {
			return command, true
		}
	}
	return nil, false
}

// Command finds only commands without profile assignments.
func (this *containerBase) Command(name string) (CommandLike, bool) {
	command, found := this.commandsByName[name]
	return command, found
}

func (this *containerBase) Flags() iter.Seq[*Flag[any, any]] {
	return slices.Values(this.flags)
}

// Flag finds only flags without profile assignments, by name or alias.
func (this *containerBase) Flag(name string) (*Flag[any, any], bool) {
	flag, found := this.flagByName[name]
	return flag, found
}

// FlagOfProfile finds an explicitly assigned flag, with no profile-free fallback.
func (this *containerBase) FlagOfProfile(name, profile string) (*Flag[any, any], bool) {
	for _, flag := range this.flagVariants[name] {
		if slices.Contains(flag.profiles, profile) {
			return flag, true
		}
	}
	return nil, false
}

// ShortFlagOfProfile finds an explicitly assigned flag by its short alias.
func (this *containerBase) ShortFlagOfProfile(short rune, profile string) (*Flag[any, any], bool) {
	for _, flag := range this.shortVariants[short] {
		if slices.Contains(flag.profiles, profile) {
			return flag, true
		}
	}
	return nil, false
}

// ShortFlag finds only flags without profile assignments.
func (this *containerBase) ShortFlag(short rune) (*Flag[any, any], bool) {
	flag, found := this.flagByShort[short]
	return flag, found
}

func (this *containerBase) Arguments() iter.Seq[*Argument[any, any]] {
	return slices.Values(this.arguments)
}

// Argument returns the definition at index, whether or not an invocation has a
// value for it. Invocation.Argument uses this same definition index.
func (this *containerBase) Argument(index int) (*Argument[any, any], bool) {
	if index < 0 || index >= len(this.arguments) {
		return nil, false
	}
	return this.arguments[index], true
}

func (this *containerBase) execute(ctx context.Context, options any) error {
	if this.handler != nil {
		return this.handler(ctx, options)
	}
	return nil
}

func (this *containerBase) close(options any) error {
	if this.closeHandler != nil {
		return this.closeHandler(options)
	}
	return nil
}
