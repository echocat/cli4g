package cli

// NoOptions is the concrete options type for definitions using only explicit
// flags and arguments. Their handlers can obtain the complete invocation and
// its values with InvocationFromContext.
type NoOptions struct{}
