package cli

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// Profiles lists all known profiles, including assignments on commands and their
// inputs, exactly once in alphabetical order. It is independent of Parse calls.
func (this *Application[O]) Profiles() iter.Seq[string] {
	return slices.Values(this.definition.profiles)
}

// ContainsProfile reports whether an exact, case-sensitive profile is known.
func (this *Application[O]) ContainsProfile(profile string) bool {
	_, found := slices.BinarySearch(this.definition.profiles, profile)
	return found
}

func validateProfiles(profiles []string) error {
	for _, profile := range profiles {
		if profile == "" || strings.Contains(profile, "|") {
			return fmt.Errorf("invalid profile name %q: must be nonempty and cannot contain |", profile)
		}
	}
	return nil
}

// Overlap means the definitions conflict even for a single selected profile.
// Disjoint assignments may coexist in a definition, but not in an active schema.
func profilesOverlap(a, b []string) bool {
	return len(a) == 0 || len(b) == 0 || slices.ContainsFunc(a, func(p string) bool { return slices.Contains(b, p) })
}

func profileActive(assignments, selected []string) bool {
	return len(assignments) == 0 || slices.ContainsFunc(assignments, func(p string) bool { return slices.Contains(selected, p) })
}

func collectProfiles(node *compileNode, known map[string]bool) {
	if command, ok := node.container.(CommandLike); ok {
		for _, profile := range command.Profiles() {
			known[profile] = true
		}
	}
	for _, flag := range node.draft.schema.flags {
		for _, profile := range flag.profiles {
			known[profile] = true
		}
	}
	for _, argument := range node.draft.schema.arguments {
		for _, profile := range argument.profiles {
			known[profile] = true
		}
	}
	for _, child := range node.children {
		collectProfiles(child, known)
	}
}
