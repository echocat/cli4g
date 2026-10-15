package cli

import (
	"context"
	"sync"
)

// Only temporary decoding state is pooled. Options, decoded values and cleanup
// live in the invocationFrame and must remain independent of scope reuse.
type parseScope struct {
	*invocationFrame
	schema    containerSchema
	positions []string
	provided  map[*Flag[any, any]]int // index in frame.flags plus one
	pooled    bool
}

var parseScopes = sync.Pool{New: func() any { return new(parseScope) }}

// frame points into the invocation's fixed backing array, not pooled storage.
func newParseScope(ctx context.Context, c Container, schema containerSchema, frame *invocationFrame) (*parseScope, error) {
	var s *parseScope
	if len(schema.flags) > 0 {
		s = parseScopes.Get().(*parseScope)
		s.pooled = true
	} else {
		// Without flags there is no index to reuse; avoid pool operations on
		// cheap positional-only scopes and command groups.
		s = new(parseScope)
	}
	*frame = invocationFrame{container: c}
	s.invocationFrame = frame
	s.schema = schema
	base := c.base()
	if base.createOptionsHandler == nil {
		frame.value = base.newOptions()
		return s, nil
	}
	value, err := base.createOptionsHandler(ctx)
	frame.value = value
	return s, err
}

func releaseParseScope(scope *parseScope) {
	if !scope.pooled {
		return
	}
	scope.reset()
	parseScopes.Put(scope)
}

func (this *parseScope) reset() {
	// Drop unusually large indexes rather than retaining their backing storage.
	index := this.provided
	if len(index) > 128 {
		index = nil
	} else {
		clear(index)
	}
	*this = parseScope{provided: index}
}

func (this *parseScope) finish(ctx context.Context, opts *ParseOpts) error {
	if err := this.finishFlags(ctx, opts); err != nil {
		return err
	}
	if err := this.parseArguments(ctx); err != nil {
		return err
	}
	return nil
}
