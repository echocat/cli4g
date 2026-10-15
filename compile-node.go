package cli

// compileNode belongs only to Create. Its container keeps its own typed facade;
// the shared compilation work does not depend on any one container's O.
type compileNode struct {
	draft     schemaDraft
	container Container
	children  []*compileNode
}
