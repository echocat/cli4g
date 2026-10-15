package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/echocat/cli4g/codec"
)

// Create compiles an independent read-only snapshot of the requirements.
// Requirements must not be modified concurrently with Create.
// Containers with subcommands cannot declare required positional arguments,
// because subcommands are recognized only before the first positional value.
func (r *ApplicationRequirement[O]) Create() (_ *Application[O], err error) {
	if r == nil || r.Name == "" {
		return nil, errSchema("application needs a name")
	}
	definition := &definitionLifecycle{fallbackToStdio: r.FallbackToStdio}
	defer func() {
		if err != nil {
			err = errors.Join(err, definition.Close())
		}
	}()
	draft, err := buildSchema(r.OptionsType(), r.Codecs, r.Flags, r.Arguments)
	if err != nil {
		return nil, fmt.Errorf("application %q: %w", r.Name, err)
	}
	if r.Help {
		flag, err := NewFlag(codec.NewBool()).WithName("help").WithShort('h').
			WithDescription("Show help.").createRaw()
		if err != nil {
			return nil, err
		}
		flag.isHelp = true
		if err := draft.schema.addFlag(flag); err != nil {
			return nil, fmt.Errorf("application %q: automatic help: %w", r.Name, err)
		}
	}
	if err := draft.codecs.AddDefaults(context.Background()); err != nil {
		return nil, err
	}
	app := &Application[O]{
		name:         r.Name,
		description:  r.Description,
		helpHandler:  r.HelpHandler,
		errorHandler: r.ErrorHandler,
		definition:   definition,
		containerBase: newContainerBase(
			r.CreateOptionsHandler,
			r.ValidateHandler,
			r.InitializeHandler,
			r.Handler,
			r.CloseHandler,
		),
	}
	if app.helpHandler == nil {
		app.helpHandler = DefaultHelpHandler
		if r.Help && app.helpHandler == nil {
			return nil, errSchema("application %q: help is enabled but DefaultHelpHandler is nil", r.Name)
		}
	}
	if r.errorHandlerDisabled {
		app.errorHandler = nil
	} else if app.errorHandler == nil {
		app.errorHandler = DefaultErrorHandler
	}
	definition.errorHandler = app.errorHandler
	root := &compileNode{draft: draft, container: app}
	active := make(map[commandRequirementIdentity]bool)
	for _, requirement := range r.Commands {
		child, err := createRequirementNode(requirement, app.commandsByName, []containerSchema{draft.schema}, active)
		if err != nil {
			return nil, err
		}
		root.children = append(root.children, child)
		command := child.container.(CommandLike)
		app.commands = append(app.commands, command)
		for _, name := range command.Names() {
			app.commandsByName[name] = command
		}
	}
	if err := compilePaths(root, nil, r.Name, definition); err != nil {
		return nil, err
	}
	return app, nil
}

// MustCreate compiles the application and panics on invalid configuration.
func (r *ApplicationRequirement[O]) MustCreate() *Application[O] {
	result, err := r.Create()
	if err != nil {
		panic(err)
	}
	return result
}

// CreateParseAndExecute manages compilation, parsing, execution and cleanup.
// Execution and completion-cleanup errors reach the error handler once,
// after cleanup. Configuration errors return directly without invoking a handler.
func (r *ApplicationRequirement[O]) CreateParseAndExecute(ctx context.Context, opts *ParseOpts, vs ...string) (err error) {
	app, err := r.Create()
	if err != nil {
		return err
	}
	defer func() {
		err = joinErrors(err, app.Close())
	}()

	i, err := app.Parse(ctx, opts, vs...)
	if err != nil {
		return err
	}
	// Definitions own no runtime resources. Keep them open while responding so
	// input-error help can still display defaults; the defer closes on return.
	return i.executeAndClose(ctx)
}

// MustCreateParseAndExecute is the panic-on-returned-error variant. Standard
// handlers exit for ordinary parse/execution failures; invalid schemas panic.
// With returning or disabled handlers, their errors also cause a panic.
func (r *ApplicationRequirement[O]) MustCreateParseAndExecute(ctx context.Context, opts *ParseOpts, vs ...string) {
	if err := r.CreateParseAndExecute(ctx, opts, vs...); err != nil {
		panic(err)
	}
}
