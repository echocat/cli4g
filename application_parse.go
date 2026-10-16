package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Parse decodes vs, resolves values, validates options, and initializes them.
// Always close a successful [Invocation], even when it is not executed.
// Failure cleans up and returns nil; the configured [ErrorHandler] may exit.
// Use [ApplicationRequirement.WithErrorHandlerEnabled] for caller-managed errors.
//
// A true --help/-h stops scanning and skips options creation and initialization.
// A handlerless target with active subcommands requires a subcommand; otherwise
// Parse returns a validation error before creating options or resolving inputs.
// Nil opts uses the OS environment; nil ctx uses context.Background.
// Callbacks must honor cancellation. The definition supports repeated and
// concurrent parsing; each invocation owns its options and resources.
// Direct Execute calls from parse hooks share this operation's error boundary.
// Context streams are available throughout parsing and the options lifecycle.
// File aliases resolve now; pass the same stream context to Execute.
func (this *Application[O]) Parse(ctx context.Context, opts *ParseOpts, vs ...string) (_ *Invocation[O], err error) {
	ctx, err = this.prepareParseContext(ctx)
	if err != nil {
		return nil, err
	}
	return this.parsePrepared(ctx, opts, vs...)
}

func (this *Application[O]) prepareParseContext(ctx context.Context) (context.Context, error) {
	if this == nil {
		return nil, fmt.Errorf("nil application")
	}
	if this.definition.closed.Load() {
		return nil, ErrApplicationClosed
	}
	return this.definition.prepareContext(ctx), nil
}

// The operation boundary prepares streams exactly once, before parsing or
// managed execution. The incoming error-boundary marker remains unchanged.
func (this *Application[O]) parsePrepared(ctx context.Context, opts *ParseOpts, vs ...string) (_ *Invocation[O], err error) {
	var input parseInput
	var profiles selectedProfiles
	if opts != nil {
		profiles = slices.Clone(opts.Profiles)
		slices.Sort(profiles)
		profiles = slices.Compact(profiles)
	}
	ctx = withProfilesContext(ctx, profiles)
	var result *Invocation[O]
	failureTarget := Container(this)
	if this.errorHandler != nil {
		responseContext := ctx
		// Registered first, so cleanup and scope release run before the handler,
		// even when a response handler exits the process rather than returning.
		defer func() {
			if err == nil {
				return
			}
			if boundary, _ := ctxValue(responseContext, errorBoundaryKey{}).(bool); boundary {
				return // the enclosing operation owns the single error response
			}
			if result == nil {
				result = this.newMetadataInvocation(input.commands)
				result.profiles = profiles
				result.selected = input.scopes
				if len(result.selected) == 0 && len(this.definition.profiles) > 0 {
					view, _ := selectContainer(this, nil, profiles)
					result.selected = []selectedContainer{view}
				}
				_ = result.Close() // no options; retain the usual failed/closed state
			}
			target := failureTarget
			if IsValidationError(err) {
				target = result.frames[len(result.frames)-1].container
			}
			err = respondError(responseContext, result, target, this.errorHandler, err)
		}()
	}
	// Nested direct execution in any parse callback belongs to this operation.
	// Keep its marker separate from the incoming context used by the response defer.
	ctx = errorContext(ctx, this.errorHandler)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, err = this.scanTokens(ctx, profiles, vs)
	if err != nil {
		return nil, err
	}
	if input.helpRequested {
		invocation := this.newHelpInvocation(input.commands)
		invocation.profiles = profiles
		invocation.selected = input.scopes
		return invocation, nil
	}
	target := input.scopes[len(input.scopes)-1]
	if target.container.base().handler == nil && len(target.commands) > 0 {
		path := []string{this.Name()}
		for _, command := range input.commands {
			path = append(path, command.Name())
		}
		return nil, NewValidationErrorf("a subcommand is required for %q", strings.Join(path, " "))
	}
	plan := this.plan
	if len(input.commands) > 0 {
		plan = input.commands[len(input.commands)-1].base().plan
	}
	scopes := make([]*parseScope, len(plan.schemas))
	hasPooledScopes := false
	defer func() {
		if !hasPooledScopes {
			return
		}
		for _, scope := range scopes {
			if scope != nil {
				releaseParseScope(scope)
			}
		}
	}()
	// Allocate all persistent frames once. This slice is never grown or pooled,
	// so scope pointers and retained invocation contexts keep stable addresses.
	frames := make([]invocationFrame, len(plan.schemas))
	// Metadata stays complete even if an earlier factory fails. Invalid values
	// represent uncreated options and are ignored by options cleanup/accessors.
	frames[0].container = this
	for i, command := range input.commands {
		frames[i+1].container = command
	}
	state := &invocationLifecycle{
		cleanup: func() error {
			var errs []error
			for i := range slices.Backward(frames) {
				frame := &frames[i]
				if frame.container != nil {
					if err := frame.close(); err != nil {
						errs = append(errs, err)
					}
				}
			}
			return errors.Join(errs...)
		},
	}
	success := false
	defer func() {
		if !success {
			err = errors.Join(err, state.close())
		}
	}()
	result = &Invocation[O]{application: this, frames: frames, lifecycle: state, selected: input.scopes, profiles: profiles}
	scopes[0], err = newParseScope(ctx, this, input.scopes[0].schema, &frames[0])
	hasPooledScopes = scopes[0].pooled
	if err != nil {
		return nil, wrapContainerError(this.Name(), "create options", err)
	}
	for i, command := range input.commands {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		failureTarget = command
		scopes[i+1], err = newParseScope(ctx, command, input.scopes[i+1].schema, &frames[i+1])
		hasPooledScopes = hasPooledScopes || scopes[i+1].pooled
		if plan.parents != nil {
			err = errors.Join(err, plan.parents.inject(ctx, i+1, frames))
		}
		if err != nil {
			return nil, wrapContainerError(command.Name(), "create options", err)
		}
	}
	if err := prepareFlagEntries(ctx, scopes, input.flags); err != nil {
		return nil, err
	}
	for _, flag := range input.flags {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		owner := scopes[flag.scope]
		failureTarget = owner.container
		definition := flag.definition
		if err := owner.addFlag(ctx, definition, definition.codec, flag.values); err != nil {
			return nil, err
		}
	}
	for i, scope := range scopes {
		failureTarget = scope.container
		scope.positions = input.positions[i]
		if err := scope.finish(ctx, opts); err != nil {
			return nil, wrapContainerError(scope.container.Name(), "", err)
		}
	}
	ctx = withInvocationContext(ctx, result)
	for _, scope := range scopes {
		failureTarget = scope.container
		if err := scope.resolve(ctx); err != nil {
			return nil, wrapContainerError(scope.container.Name(), "resolve", err)
		}
	}
	for _, scope := range scopes {
		failureTarget = scope.container
		if err := scope.validate(ctx); err != nil {
			return nil, wrapContainerError(scope.container.Name(), "validate", err)
		}
	}
	for _, scope := range scopes {
		failureTarget = scope.container
		if err := scope.initialize(ctx); err != nil {
			return nil, wrapContainerError(scope.container.Name(), "initialize", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state.ready = true
	success = true
	return result, nil
}

// Input messages already name the affected flag/argument; the help view supplies
// application/command context. Keep phase/container details for operational errors.
func wrapContainerError(name, phase string, err error) error {
	if err == nil || IsValidationError(err) {
		return err
	}
	if phase == "" {
		return fmt.Errorf("%s: %w", name, err)
	}
	return fmt.Errorf("%s: %s: %w", name, phase, err)
}

// MustParse calls [Application.Parse] and panics on a returned error.
func (this *Application[O]) MustParse(ctx context.Context, opts *ParseOpts, vs ...string) *Invocation[O] {
	result, err := this.Parse(ctx, opts, vs...)
	if err != nil {
		panic(err)
	}
	return result
}

// ParseAndExecute parses, executes, and closes an invocation.
// The application remains reusable until [Application.Close].
func (this *Application[O]) ParseAndExecute(ctx context.Context, opts *ParseOpts, vs ...string) (err error) {
	ctx, err = this.prepareParseContext(ctx)
	if err != nil {
		return err
	}
	i, err := this.parsePrepared(ctx, opts, vs...)
	if err != nil {
		return err
	}
	return i.executeAndClose(ctx)
}

// MustParseAndExecute calls [Application.ParseAndExecute] and panics on a returned error.
func (this *Application[O]) MustParseAndExecute(ctx context.Context, opts *ParseOpts, vs ...string) {
	if err := this.ParseAndExecute(ctx, opts, vs...); err != nil {
		panic(err)
	}
}
