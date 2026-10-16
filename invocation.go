package cli

import (
	"context"
	"io"
	"iter"
	"slices"
)

// InvocationLike gives handlers and tools access to any invocation's metadata
// and decoded values. It is implemented by [Invocation], not by callers.
type InvocationLike interface {
	Root() Container
	HelpRequested() bool
	HelpInfo() HelpInfo
	HelpHandler() HelpHandler
	ErrorHandler() ErrorHandler
	Stack() []InvocationStackFrame
	NameStack() []string
	Profiles() iter.Seq[string]
	ContainsProfile(string) bool
	Flags() iter.Seq2[*Flag[any, any], any]
	Flag(string) (*Flag[any, any], any)
	ShortFlag(rune) (*Flag[any, any], any)
	Arguments() iter.Seq2[*Argument[any, any], any]
	Argument(int) (*Argument[any, any], any)
	createOptions(CommandLike) (any, bool)
	optionsFor(Container) (any, bool)
}

// Invocation owns the options and resources of one successful parse.
// Execute it with [Invocation.Execute] and always call [Invocation.Close].
// Option getters read current fields; input getters retain decoded values.
// Pointers, maps, and slices can share data within this invocation.
type Invocation[O any] struct {
	application *Application[O]
	frames      []invocationFrame // fixed storage: root application followed by selected commands
	lifecycle   *invocationLifecycle
	selected    []selectedContainer // retained active schema, including help/error invocations
	profiles    selectedProfiles
}

var _ io.Closer = (*Invocation[NoOptions])(nil)

// Application returns the application and its current options. Struct options
// are returned by value; pointer options refer to the owned instance. Reference
// fields in returned structs still share their underlying data.
// Help invocations have no options and return the zero value of O.
func (this Invocation[O]) Application() (*Application[O], O) {
	if len(this.frames) == 0 || !this.frames[0].value.IsValid() {
		var empty O
		return this.application, empty
	}
	return this.application, this.frames[0].currentOptions().(O)
}

// Root exposes the application's metadata when its concrete options type is
// unknown to a command handler obtaining the invocation from its context.
func (this Invocation[O]) Root() Container {
	return this.application
}

// Profiles lists this invocation's selected profiles exactly once in alphabetical
// order. Unlike Application.Profiles, it does not list every known profile.
// The snapshot remains readable after Close and on help/error invocations.
func (this Invocation[O]) Profiles() iter.Seq[string] { return this.profiles.Profiles() }

// ContainsProfile reports whether an exact, case-sensitive profile was selected
// for this invocation, independently of whether the command path uses it.
func (this Invocation[O]) ContainsProfile(profile string) bool {
	return this.profiles.ContainsProfile(profile)
}

// InvocationStackFrame contains a selected command and its current options.
// Use [Command.OptionsFrom] for typed access. Reference data remains shared.
type InvocationStackFrame struct {
	Command CommandLike
	Options any
}

// Stack returns a new list of the selected commands and their current options.
// It does not deep-copy options or include the root application.
// Help invocations contain the selected commands with nil Options.
func (this Invocation[O]) Stack() []InvocationStackFrame {
	frames := this.commandFrames()
	if len(frames) == 0 {
		return nil
	}
	stack := make([]InvocationStackFrame, len(frames))
	for i := range frames {
		frame := &frames[i]
		stack[i] = InvocationStackFrame{
			Command: frame.container.(CommandLike),
			Options: frame.currentOptions(),
		}
	}
	return stack
}

func (this Invocation[O]) commandFrames() []invocationFrame {
	if len(this.frames) == 0 {
		return nil
	}
	return this.frames[1:]
}

// NameStack returns the selected canonical command names, excluding the application.
func (this Invocation[O]) NameStack() []string {
	frames := this.commandFrames()
	names := make([]string, len(frames))
	for i := range frames {
		names[i] = frames[i].container.Name()
	}
	return names
}

// Flags exposes successfully resolved runtime values, including after failed
// parsing or cleanup. Prepared requirements are never published by these getters.
func (this Invocation[O]) Flags() iter.Seq2[*Flag[any, any], any] {
	return func(yield func(*Flag[any, any], any) bool) {
		for i := range this.frames {
			frame := &this.frames[i]
			for f, value := range frame.eachFlag() {
				if !yield(f, value) {
					return
				}
			}
		}
	}
}

// Flag returns a decoded flag by name or alias; the deepest selected definition wins.
// Missing values return nil, nil. Factory-only fields are not decoded inputs.
func (this Invocation[O]) Flag(name string) (*Flag[any, any], any) {
	for i := range slices.Backward(this.frames) {
		if f, value := this.frames[i].flag(name); f != nil {
			return f, value
		}
	}
	return nil, nil
}

// ShortFlag returns a decoded flag by its short alias, or nil, nil if absent.
func (this Invocation[O]) ShortFlag(short rune) (*Flag[any, any], any) {
	for i := range slices.Backward(this.frames) {
		if f, value := this.frames[i].shortFlag(short); f != nil {
			return f, value
		}
	}
	return nil, nil
}

// Arguments and Argument expose resolved runtime values of the deepest selected container.
// Arguments iterates only values present in the invocation. Argument uses the
// container's definition index; omitted arguments return nil, nil.
func (this Invocation[O]) Arguments() iter.Seq2[*Argument[any, any], any] {
	if len(this.frames) == 0 {
		return invocationFrame{}.eachArgument()
	}
	return this.frames[len(this.frames)-1].eachArgument()
}

// Argument returns a decoded leaf argument by definition index, or nil, nil if absent.
func (this Invocation[O]) Argument(i int) (*Argument[any, any], any) {
	if len(this.frames) == 0 {
		return nil, nil
	}
	return this.frames[len(this.frames)-1].argument(i)
}

// Execute runs handlers from root to leaf and checks cancellation before and
// after each handler. Handlers must honor their context during their own work.
// A nil context is replaced with context.Background. With a configured error
// handler, failures close this invocation before reporting. Without one,
// the caller retains cleanup responsibility. Successful invocations must be closed.
// A help invocation executes only the application's HelpHandler.
// Context streams apply to handlers; file values already resolved by Parse
// retain their original streams. Use the same stream context for both phases.
func (this *Invocation[O]) Execute(ctx context.Context) (err error) {
	if this == nil || this.lifecycle == nil || !this.lifecycle.ready {
		return ErrInvocationNotReady
	}
	ctx = this.application.definition.prepareContext(ctx)
	target := Container(this.application)
	handler := this.ErrorHandler()
	defer func() {
		if err != nil && handler != nil {
			err = joinErrors(err, this.Close())
			if boundary, _ := ctxValue(ctx, errorBoundaryKey{}).(bool); !boundary {
				err = respondError(ctx, this, target, handler, err)
			}
		}
	}()
	return this.execute(errorContext(ctx, handler), &target)
}

// ErrorHandler returns the application's captured input/operational response.
func (this Invocation[O]) ErrorHandler() ErrorHandler {
	if this.application == nil {
		return nil
	}
	return this.application.errorHandler
}

func (this *Invocation[O]) executeAndClose(ctx context.Context) (err error) {
	ctx = this.application.definition.prepareContext(ctx)
	handler := this.ErrorHandler()
	target := Container(this.application)
	closed := false
	defer func() {
		if !closed {
			err = joinErrors(err, this.Close())
		}
	}()
	err = this.execute(errorContext(ctx, handler), &target)
	err = joinErrors(err, this.Close())
	closed = true
	if boundary, _ := ctxValue(ctx, errorBoundaryKey{}).(bool); boundary {
		return err
	}
	return respondError(ctx, this, target, handler, err)
}

// execute leaves reporting and cleanup to its enclosing execution boundary.
func (this *Invocation[O]) execute(ctx context.Context, target *Container) error {
	if this == nil || this.lifecycle == nil || !this.lifecycle.ready {
		return ErrInvocationNotReady
	}
	if this.lifecycle.closed.Load() {
		return ErrInvocationClosed
	}
	if this.application.definition.closed.Load() {
		return ErrApplicationClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withInvocationContext(ctx, this)
	if err := ctx.Err(); err != nil {
		return err
	}
	if this.HelpRequested() {
		*target = this.frames[len(this.frames)-1].container
		if err := this.application.helpHandler(withContainerContext(ctx, *target), this); err != nil {
			return err
		}
		return ctx.Err()
	}
	for i := range this.frames {
		frame := &this.frames[i]
		*target = frame.container
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := frame.container.base().execute(withContainerContext(ctx, frame.container), frame.currentOptions()); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Close closes owned options in reverse scope order, exactly once. Repeated
// calls return the original cleanup result. Callers must Close every invocation
// returned by a successful Parse, including after an Execute error.
// Execute and Close must not run concurrently on the same invocation.
func (this *Invocation[O]) Close() error {
	if this == nil || this.lifecycle == nil {
		return nil
	}
	return this.lifecycle.close()
}

func (this Invocation[O]) createOptions(command CommandLike) (any, bool) {
	return this.optionsFor(command)
}
func (this Invocation[O]) optionsFor(container Container) (any, bool) {
	for i := range this.frames {
		frame := &this.frames[i]
		if frame.container == container {
			if !frame.value.IsValid() {
				return nil, false
			}
			return frame.currentOptions(), true
		}
	}
	return nil, false
}
