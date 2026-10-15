package cli

import (
	"context"
	"os"
)

// ParseOpts customizes environment input. Nil uses os.LookupEnv.
type ParseOpts struct {
	// LookupEnv replaces os.LookupEnv. Return exists=false to disable environment input.
	// CLI flags take precedence; callbacks used by concurrent parses must be safe.
	LookupEnv func(ctx context.Context, name string) (value string, exists bool)
}

func (this *ParseOpts) lookupEnv(ctx context.Context, name string) (string, bool) {
	if this != nil && this.LookupEnv != nil {
		return this.LookupEnv(ctx, name)
	}
	return os.LookupEnv(name)
}
