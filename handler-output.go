package cli

import (
	"context"
	"io"
	"sync"

	"github.com/echocat/cli4g/codec"
)

func handlerOutput(ctx context.Context, configured io.Writer) io.Writer {
	if configured != nil {
		return configured
	}
	if output := codec.StderrOfContext(ctx); output != nil {
		return output
	}
	return io.Discard
}

// All framework help/error writes share this short-lived lock. Independent
// handlers may use the same non-concurrent writer. Rendering and callbacks do
// not run under this lock, so response -> help cannot recursively lock it.
// Custom/direct writer access still needs the caller's synchronization.
var handlerOutputMutex sync.Mutex
