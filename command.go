package cli

import (
	"context"
	"slices"
)

// CommandLike allows commands with different options types in one command tree.
type CommandLike interface {
	Container
	isCommand()
}

// Command is a typed command definition belonging to an [Application].
// Build it with [NewCommand]. Read selected options with [Command.OptionsFrom].
type Command[O any] struct {
	names       []string
	description string
	*containerBase
}

func (*Command[O]) isCommand() {}

// Name returns the primary command name.
func (c *Command[O]) Name() string {
	return c.names[0]
}

// Names returns a new slice containing the primary name and aliases.
func (c *Command[O]) Names() []string {
	return slices.Clone(c.names)
}

// Description returns the help description.
func (c *Command[O]) Description() string {
	return c.description
}

// Validate runs the options' Validate method followed by the configured callback.
func (c *Command[O]) Validate(ctx context.Context, options O) error {
	if err := c.containerBase.requireOpen(); err != nil {
		return err
	}
	frame := directOptionsFrame(c, options)
	return frame.validate(withContainerContext(ctx, c))
}

// Initialize runs options methods and callbacks. Use pointer options to retain
// mutations in the caller's instance; struct arguments follow Go value semantics.
func (c *Command[O]) Initialize(ctx context.Context, options O) error {
	if err := c.containerBase.requireOpen(); err != nil {
		return err
	}
	frame := directOptionsFrame(c, options)
	return frame.initialize(withContainerContext(ctx, c))
}

// Execute runs this command's handler with caller-owned options.
// It does not parse, validate, initialize, or close those options.
func (c *Command[O]) Execute(ctx context.Context, options O) error {
	return c.containerBase.executeDirect(ctx, c, options)
}

// Close runs the close callback and options' Close method, including partial state.
func (c *Command[O]) Close(options O) error {
	frame := directOptionsFrame(c, options)
	return frame.close()
}

// OptionsFrom reads this command's options from an invocation.
// Unselected commands and help invocations return false. Structs are copied;
// pointers and reference fields share invocation-owned data.
func (c *Command[O]) OptionsFrom(invocation InvocationLike) (O, bool) {
	var empty O
	if c == nil || invocation == nil {
		return empty, false
	}
	options, found := invocation.createOptions(c)
	if !found {
		return empty, false
	}
	typed, ok := options.(O)
	return typed, ok
}
