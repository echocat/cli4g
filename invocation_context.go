package cli

import "context"

type invocationContextKey struct{}

// InvocationFromContext returns the complete invocation associated with a
// lifecycle hook or handler. No new invocation is attached during decoding.
func InvocationFromContext(ctx context.Context) (InvocationLike, bool) {
	if ctx == nil {
		return nil, false
	}
	invocation, found := ctx.Value(invocationContextKey{}).(InvocationLike)
	return invocation, found
}

func withInvocationContext(ctx context.Context, invocation InvocationLike) context.Context {
	return withProfilesContext(ctx, invocation)
}
