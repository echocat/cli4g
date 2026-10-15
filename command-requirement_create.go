package cli

import (
	"fmt"
	"slices"
)

func (r CommandRequirement[O]) createNode(
	siblings map[string]CommandLike,
	parents []containerSchema,
	active map[commandRequirementIdentity]bool,
) (*compileNode, error) {
	if len(r.Names) == 0 || r.Names[0] == "" {
		return nil, errSchema("command needs a name")
	}
	seen := make(map[string]bool)
	for _, name := range r.Names {
		if name == "" || name[0] == '-' || siblings[name] != nil || seen[name] {
			return nil, errSchema("duplicate or invalid command name %q", name)
		}
		seen[name] = true
	}
	draft, err := buildSchema(r.OptionsType(), r.Codecs, r.Flags, r.Arguments)
	if err != nil {
		return nil, fmt.Errorf("command %q: %w", r.Names[0], err)
	}
	for _, parent := range parents {
		for name := range draft.schema.flagByName {
			if parent.flagByName[name] != nil {
				return nil, errSchema("command %q: inherited flag %q is duplicated", r.Names[0], name)
			}
		}
		for short := range draft.schema.flagByShort {
			if parent.flagByShort[short] != nil {
				return nil, errSchema("command %q: inherited short flag %q is duplicated", r.Names[0], short)
			}
		}
	}
	command := &Command[O]{
		names:       slices.Clone(r.Names),
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
		child, err := createRequirementNode(requirement, command.commandsByName, append(slices.Clone(parents), draft.schema), active)
		if err != nil {
			return nil, err
		}
		node.children = append(node.children, child)
		childCommand := child.container.(CommandLike)
		command.commands = append(command.commands, childCommand)
		for _, name := range childCommand.Names() {
			command.commandsByName[name] = childCommand
		}
	}
	return node, nil
}
