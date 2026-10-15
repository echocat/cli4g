package cli

import (
	"context"
	"io"
)

var _ io.Closer = (*Application[NoOptions])(nil)

// Application is a reusable CLI definition with options of type O.
// Obtain it from [ApplicationRequirement.Create]; close invocations before
// [Application.Close]. Concurrent parses require concurrent-safe user callbacks.
type Application[O any] struct {
	name         string
	description  string
	helpHandler  HelpHandler
	errorHandler ErrorHandler
	definition   *definitionLifecycle
	*containerBase
}

// Name returns the application's command-line name.
func (this *Application[O]) Name() string {
	return this.name
}

// Names returns a new slice containing [Application.Name].
func (this *Application[O]) Names() []string {
	return []string{this.name}
}

// Description returns the help description.
func (this *Application[O]) Description() string {
	return this.description
}

// Validate runs the options' Validate method followed by the configured callback.
func (this *Application[O]) Validate(ctx context.Context, options O) error {
	if err := this.containerBase.requireOpen(); err != nil {
		return err
	}
	frame := directOptionsFrame(this, options)
	return frame.validate(withContainerContext(ctx, this))
}

// Initialize runs options methods and callbacks like Parse. Use pointer options
// when assignments must be retained by the caller; struct arguments are copies.
func (this *Application[O]) Initialize(ctx context.Context, options O) error {
	if err := this.containerBase.requireOpen(); err != nil {
		return err
	}
	frame := directOptionsFrame(this, options)
	return frame.initialize(withContainerContext(ctx, this))
}

// Execute runs the application handler with caller-owned options.
// It does not parse, validate, initialize, or close those options.
// Use [Application.ParseAndExecute] for the complete CLI lifecycle.
func (this *Application[O]) Execute(ctx context.Context, options O) error {
	return this.containerBase.executeDirect(ctx, this, options)
}

// CloseOptions runs the close callback and options' Close method, including for
// partially initialized caller-owned options.
func (this *Application[O]) CloseOptions(options O) error {
	frame := directOptionsFrame(this, options)
	return frame.close()
}

// Close prevents further parsing and execution, including through commands.
// Close all invocations first. Repeated calls are safe; metadata stays readable.
func (this *Application[O]) Close() error {
	if this == nil {
		return nil
	}
	return this.definition.Close()
}
