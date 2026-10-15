package cli

import "reflect"

// CommandRequirementLike accepts value and pointer builders with independent
// options types. Each builder compiles its own typed handler adapters.
type CommandRequirementLike interface {
	createNode(map[string]CommandLike, []containerSchema, map[commandRequirementIdentity]bool) (*compileNode, error)
	commandsIdentity() commandRequirementIdentity
}

// Shared child slices can form cycles through pointer or value builders alike.
// Identity is used only along the active compile path, so a builder can still
// be reused by separate application definitions or branches.
type commandRequirementIdentity struct {
	typ      reflect.Type
	children uintptr
	count    int
}

func (r CommandRequirement[O]) commandsIdentity() commandRequirementIdentity {
	if len(r.Commands) == 0 {
		return commandRequirementIdentity{}
	}
	return commandRequirementIdentity{
		typ:      reflect.TypeFor[CommandRequirement[O]](),
		children: reflect.ValueOf(r.Commands).Pointer(),
		count:    len(r.Commands),
	}
}

func createRequirementNode(requirement CommandRequirementLike, siblings map[string]CommandLike, parents []containerSchema, active map[commandRequirementIdentity]bool) (*compileNode, error) {
	if requirement == nil {
		return nil, errSchema("nil command requirement")
	}
	value := reflect.ValueOf(requirement)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, errSchema("nil command requirement")
	}
	identity := requirement.commandsIdentity()
	if identity.children != 0 {
		if active[identity] {
			return nil, errSchema("cyclic command requirement")
		}
		active[identity] = true
		defer delete(active, identity)
	}
	return requirement.createNode(siblings, parents, active)
}
