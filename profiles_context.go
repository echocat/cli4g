package cli

import (
	"context"
	"iter"
	"slices"
)

type profilesContextKey struct{}

type profileSource interface {
	Profiles() iter.Seq[string]
	ContainsProfile(string) bool
}

// A single immutable overlay carries either the early parse selection or the
// complete invocation. Retained contexts never observe a later replacement.
type invocationMetadataContext struct {
	context.Context
	source profileSource
}

func (this *invocationMetadataContext) Value(key any) any {
	switch key.(type) {
	case profilesContextKey:
		return this.source
	case invocationContextKey:
		if invocation, ok := this.source.(InvocationLike); ok {
			return invocation
		}
	}
	return this.Context.Value(key)
}

// selectedProfiles is a sorted, deduplicated snapshot owned by the parse and its
// invocation. No getter exposes its mutable backing array.
type selectedProfiles []string

func (profiles selectedProfiles) Profiles() iter.Seq[string] { return slices.Values(profiles) }

func (profiles selectedProfiles) ContainsProfile(profile string) bool {
	_, found := slices.BinarySearch(profiles, profile)
	return found
}

// ProfilesFromContext returns the selected profiles of the current parse or
// invocation, alphabetically and without duplicates. It is available throughout
// parsing, including options factories and decoding, and in lifecycle, help,
// execution, and error callbacks. A parse with no selected profiles returns an
// empty sequence and true. Missing context data returns an empty sequence and
// false. Parse errors expose the requested selection, including unknown names.
func ProfilesFromContext(ctx context.Context) (iter.Seq[string], bool) {
	if profiles, ok := ctxValue(ctx, profilesContextKey{}).(profileSource); ok {
		return profiles.Profiles(), true
	}
	return selectedProfiles(nil).Profiles(), false
}

// ContextContainsProfile reports whether profile was selected for the current
// parse or invocation. Missing context data returns false. Names are exact and
// case-sensitive; profile-free definitions do not add implicit selections.
func ContextContainsProfile(ctx context.Context, profile string) bool {
	profiles, ok := ctxValue(ctx, profilesContextKey{}).(profileSource)
	return ok && profiles.ContainsProfile(profile)
}

func withProfilesContext(ctx context.Context, profiles profileSource) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &invocationMetadataContext{Context: ctx, source: profiles}
}
