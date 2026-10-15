package cli

import "context"

type containerContextKey struct{}

// ContainerFromContext identifies the application or command whose hook or
// handler is currently running, independently of its options type.
func ContainerFromContext(ctx context.Context) (Container, bool) {
	if ctx == nil {
		return nil, false
	}
	container, found := ctx.Value(containerContextKey{}).(Container)
	return container, found
}

func withContainerContext(ctx context.Context, container Container) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, containerContextKey{}, container)
}
