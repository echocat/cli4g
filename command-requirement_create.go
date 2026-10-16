package cli

import (
	"fmt"
	"slices"
)

type commandDepthError struct {
	path    string
	maximum uint8
}

func (e *commandDepthError) Error() string {
	return fmt.Sprintf("maximum command depth %d exceeded at %q", e.maximum, e.path)
}

func (r CommandRequirement[O]) createNode(
	siblings []CommandLike,
	parents []containerSchema,
	active map[commandRequirementIdentity]bool,
	maxDepth uint8,
) (*compileNode, error) {
	if len(r.Names) == 0 || r.Names[0] == "" {
		return nil, errSchema("command needs a name")
	}
	if len(parents) > int(maxDepth) {
		return nil, &commandDepthError{path: r.Names[0], maximum: maxDepth}
	}
	if err := validateProfiles(r.Profiles); err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, name := range r.Names {
		if name == "" || name[0] == '-' || seen[name] {
			return nil, errSchema("duplicate or invalid command name %q", name)
		}
		for _, sibling := range siblings {
			if slices.Contains(sibling.Names(), name) && profilesOverlap(sibling.Profiles(), r.Profiles) {
				return nil, errSchema("duplicate or invalid command name %q (profiles %v and %v)", name, sibling.Profiles(), r.Profiles)
			}
		}
		seen[name] = true
	}
	draft, err := buildSchema(r.OptionsType(), r.Codecs, r.Flags, r.Arguments)
	if err != nil {
		return nil, fmt.Errorf("command %q: %w", r.Names[0], err)
	}
	for _, parent := range parents {
		for _, flag := range draft.schema.flags {
			for _, inherited := range parent.flags {
				if profilesOverlap(flag.profiles, inherited.profiles) {
					if err := conflictingFlags(inherited, flag); err != nil {
						return nil, errSchema("command %q: inherited %v", r.Names[0], err)
					}
				}
			}
		}
	}
	command := &Command[O]{
		names:       slices.Clone(r.Names),
		profiles:    slices.Clone(r.Profiles),
		description: r.Description,
		visible:     r.Visible,
		containerBase: newContainerBase(
			r.CreateOptionsHandler,
			r.ValidateHandler,
			r.InitializeHandler,
			r.Handler,
			r.CloseHandler,
		),
	}
	node := &compileNode{draft: draft, container: command}
	for _, requirement := range r.Commands {
		child, err := createRequirementNode(requirement, command.commands, append(slices.Clone(parents), draft.schema), active, maxDepth)
		if err != nil {
			if depth, ok := err.(*commandDepthError); ok {
				depth.path = r.Names[0] + " " + depth.path
			}
			return nil, err
		}
		node.children = append(node.children, child)
		childCommand := child.container.(CommandLike)
		command.commands = append(command.commands, childCommand)
		if len(childCommand.Profiles()) == 0 {
			for _, name := range childCommand.Names() {
				command.commandsByName[name] = childCommand
			}
		}
	}
	return node, nil
}
