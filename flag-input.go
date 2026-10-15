package cli

type flagInput struct {
	scope      int
	definition *Flag[any, any]
	values     []string
}
