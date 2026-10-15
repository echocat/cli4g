package cli

import (
	"context"
	"sync"
)

// HelpHandler executes a valid help invocation. Its metadata includes the
// selected path's resolved codecs; no options or normal lifecycle are created.
type HelpHandler func(context.Context, InvocationLike) error

var defaultHelpHandler = sync.OnceValue(func() HelpHandler {
	return (&TemplatedHelpHandlerRequirement{
		ApplicationTemplate:  DefaultApplicationHelpTemplate,
		CommandTemplate:      DefaultCommandHelpTemplate,
		UsageTemplate:        DefaultUsageHelpTemplate,
		CommandEntryTemplate: DefaultCommandEntryHelpTemplate,
		FlagTemplate:         DefaultFlagHelpTemplate,
		ArgumentTemplate:     DefaultArgumentHelpTemplate,
	}).MustCreate()
})

// DefaultHelpHandler is the overridable fallback captured by ApplicationRequirement.Create
// when no application handler is set. Configure it before Create, without concurrent
// modification. Its built-in value compiles templates once on first use, writes
// to os.Stderr and returns normally.
var DefaultHelpHandler HelpHandler = func(ctx context.Context, invocation InvocationLike) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return defaultHelpHandler()(ctx, invocation)
}
