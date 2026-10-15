package cli

type parsePlan struct {
	schemas    []containerSchema
	definition *definitionLifecycle
	parents    *parentBindings // nil on paths without parent injection
}
