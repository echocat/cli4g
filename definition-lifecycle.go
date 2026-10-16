package cli

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/echocat/cli4g/codec"
)

var ErrApplicationClosed = errors.New("application is closed")

// Definitions contain immutable resource-free requirements. Closing invalidates
// new operations; it needs no resource leases, release callbacks or reader locks.
type definitionLifecycle struct {
	closed          atomic.Bool
	errorHandler    ErrorHandler
	fallbackToStdio bool
	profiles        []string
	maxCommandDepth uint8
	commandDepth    uint8 // deepest compiled subcommand level, application excluded
}

func (this *definitionLifecycle) prepareContext(ctx context.Context) context.Context {
	return codec.ContextWithFallbackToStdio(ctx, this != nil && this.fallbackToStdio)
}

func (this *definitionLifecycle) beginRead() error {
	if this != nil && this.closed.Load() {
		return ErrApplicationClosed
	}
	return nil
}
func (*definitionLifecycle) endRead() {}
func (this *definitionLifecycle) Close() error {
	if this != nil {
		this.closed.Store(true)
	}
	return nil
}
