package cli

import (
	"fmt"
	"slices"
)

// selectedContainer references an immutable compiled or invocation-local schema.
// Public metadata always exposes the complete immutable definition, never a
// mutable active-profile switch.
type selectedContainer struct {
	container     Container
	schema        *containerSchema
	commands      []CommandLike
	commandByName map[string]CommandLike
}

func conflictingFlags(a, b *Flag[any, any]) error {
	for _, name := range b.names {
		if slices.Contains(a.names, name) {
			return fmt.Errorf("flag %q is duplicated (definitions --%s profiles %v and --%s profiles %v)", name, a.Name(), a.profiles, b.Name(), b.profiles)
		}
	}
	for _, short := range b.shorts {
		if slices.Contains(a.shorts, short) {
			return fmt.Errorf("short flag %q is duplicated (definitions --%s profiles %v and --%s profiles %v)", short, a.Name(), a.profiles, b.Name(), b.profiles)
		}
	}
	return nil
}

func selectSchema(source *containerSchema, profiles []string) (*containerSchema, error) {
	if !source.hasProfiles {
		return source, nil
	}
	if len(profiles) == 0 && source.unprofiled != nil {
		return source.unprofiled, nil
	}
	result := &containerSchema{optionsType: source.optionsType,
		flagByName: make(map[string]*Flag[any, any]), flagByShort: make(map[rune]*Flag[any, any])}
	var bindings []*fieldBinding
	addBinding := func(binding *fieldBinding) error {
		if binding == nil {
			return nil
		}
		for _, previous := range bindings {
			if fieldBindingsOverlap(previous, binding) {
				return fmt.Errorf("overlapping options field binding for %v and %v", previous.path, binding.path)
			}
		}
		bindings = append(bindings, binding)
		return nil
	}
	for _, flag := range source.flags {
		if !profileActive(flag.profiles, profiles) {
			continue
		}
		for _, name := range flag.names {
			if previous := result.flagByName[name]; previous != nil {
				return result, conflictingFlags(previous, flag)
			}
			result.flagByName[name] = flag
		}
		for _, short := range flag.shorts {
			if previous := result.flagByShort[short]; previous != nil {
				return result, conflictingFlags(previous, flag)
			}
			result.flagByShort[short] = flag
		}
		if err := addBinding(flag.binding); err != nil {
			return result, fmt.Errorf("flag --%s profiles %v: %w", flag.Name(), flag.profiles, err)
		}
		result.flags = append(result.flags, flag)
	}
	for _, argument := range source.arguments {
		if !profileActive(argument.profiles, profiles) {
			continue
		}
		for _, previous := range result.arguments {
			if previous.name == argument.name {
				return result, fmt.Errorf("argument %q is duplicated (profiles %v and %v)", argument.name, previous.profiles, argument.profiles)
			}
			if previous.arity != (valueArity{min: 1, max: 1}) {
				return result, fmt.Errorf("argument %s: only the final argument may be variadic (profiles %v and %v)", previous.name, previous.profiles, argument.profiles)
			}
		}
		if err := addBinding(argument.binding); err != nil {
			return result, fmt.Errorf("argument %s profiles %v: %w", argument.name, argument.profiles, err)
		}
		result.arguments = append(result.arguments, argument)
	}
	return result, nil
}

func fieldBindingsOverlap(a, b *fieldBinding) bool {
	for i := 0; i < min(len(a.steps), len(b.steps)); i++ {
		if a.steps[i].index != b.steps[i].index {
			return false
		}
	}
	return true
}

func selectContainer(container Container, parents []selectedContainer, profiles []string) (selectedContainer, error) {
	base := container.base()
	if len(profiles) == 0 {
		// Create validated and retained this immutable selection, including on
		// definitions that also contain profile-assigned inputs or commands.
		return base.unprofiled, nil
	}
	result := selectedContainer{container: container}
	var err error
	result.schema, err = selectSchema(&base.containerSchema, profiles)
	if err != nil {
		return result, err
	}
	if len(base.unprofiled.commands) == len(base.commands) {
		result.commands, result.commandByName = base.commands, base.commandsByName
	} else {
		result.commandByName = make(map[string]CommandLike)
		for _, command := range base.commands {
			if !profileActive(command.Profiles(), profiles) {
				continue
			}
			for _, name := range command.Names() {
				if previous := result.commandByName[name]; previous != nil {
					return result, fmt.Errorf("command %q is duplicated (definitions %s profiles %v and %s profiles %v)", name, previous.Name(), previous.Profiles(), command.Name(), command.Profiles())
				}
				result.commandByName[name] = command
			}
			result.commands = append(result.commands, command)
		}
	}
	if len(result.commands) > 0 {
		for _, argument := range result.schema.arguments {
			if argument.required {
				return result, fmt.Errorf("required positional argument %q cannot be combined with subcommands", argument.name)
			}
		}
	}
	for _, parent := range parents {
		for _, inherited := range parent.schema.flags {
			for _, flag := range result.schema.flags {
				if err := conflictingFlags(inherited, flag); err != nil {
					return result, fmt.Errorf("inherited %w", err)
				}
			}
		}
	}
	return result, nil
}
