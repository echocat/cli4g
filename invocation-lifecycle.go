package cli

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrInvocationNotReady = errors.New("invocation is not initialized")
	ErrInvocationClosed   = errors.New("invocation is already closed")
)

// invocationLifecycle belongs to one Parse result and is shared by its copies,
// never by the compiled application or a different invocation.
type invocationLifecycle struct {
	ready         bool
	helpRequested bool
	closed        atomic.Bool
	once          sync.Once
	cleanup       func() error
	operation     func(context.Context) error // request-only protocol invocation
	err           error
}

func (state *invocationLifecycle) close() error {
	if state == nil {
		return nil
	}
	state.once.Do(func() {
		state.closed.Store(true)
		state.err = state.cleanup()
	})
	return state.err
}
