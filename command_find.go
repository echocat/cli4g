package cli

// FindCommand resolves a command path and verifies its options type once.
// The returned handle provides typed access to that command's invocation options.
func FindCommand[O any](container Container, path ...string) (*Command[O], bool) {
	if container == nil || len(path) == 0 {
		return nil, false
	}
	var command CommandLike
	for _, name := range path {
		var found bool
		command, found = container.Command(name)
		if !found {
			return nil, false
		}
		container = command
	}
	typed, ok := command.(*Command[O])
	return typed, ok
}

// FindCommandOfProfile resolves a path for one profile and verifies its leaf
// options type. Explicit assignments are preferred; profile-free commands are
// also traversed. It does not activate ancestors assigned to other profiles.
func FindCommandOfProfile[O any](container Container, profile string, path ...string) (*Command[O], bool) {
	if container == nil || len(path) == 0 {
		return nil, false
	}
	var command CommandLike
	for _, name := range path {
		var found bool
		command, found = container.CommandOfProfile(name, profile)
		if !found {
			command, found = container.Command(name)
		}
		if !found {
			return nil, false
		}
		container = command
	}
	typed, ok := command.(*Command[O])
	return typed, ok
}
