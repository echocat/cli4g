package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// Create compiles an independent template set. Template execution may run
// concurrently; framework help/error writes are serialized and rendering errors
// do not produce partial output. Custom functions must be concurrency-safe.
func (this *TemplatedHelpHandlerRequirement) Create() (HelpHandler, error) {
	if this == nil {
		return nil, fmt.Errorf("nil templated help handler requirement")
	}
	funcs := template.FuncMap{"join": strings.Join, "quote": strconv.Quote}
	for name, function := range this.Funcs {
		funcs[name] = function
	}
	// Funcs panics for invalid function signatures; convert configuration errors
	// into Create errors, consistent with template syntax errors.
	templates, err := createHelpTemplates(funcs, []helpTemplateSource{
		{"application", this.ApplicationTemplate},
		{"command", this.CommandTemplate},
		{"usage", this.UsageTemplate},
		{"command-entry", this.CommandEntryTemplate},
		{"flag", this.FlagTemplate},
		{"argument", this.ArgumentTemplate},
	})
	if err != nil {
		return nil, err
	}
	output := this.Output
	exitCode := this.ExitCode
	return func(ctx context.Context, invocation InvocationLike) error {
		if ctx == nil {
			ctx = context.Background()
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := buildHelpTemplateData(ctx, invocation)
		if err != nil {
			return err
		}
		name := "command"
		if len(data.Path) == 1 {
			name = "application"
		}
		var buffer bytes.Buffer
		if err := templates.ExecuteTemplate(&buffer, name, data); err != nil {
			return fmt.Errorf("render %s help: %w", name, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		err = func() error {
			handlerOutputMutex.Lock()
			defer handlerOutputMutex.Unlock()
			if err := ctx.Err(); err != nil {
				return err
			}
			n, err := handlerOutput(ctx, output).Write(buffer.Bytes())
			if err != nil {
				return fmt.Errorf("write help: %w", err)
			}
			if n != buffer.Len() {
				return fmt.Errorf("write help: %w", io.ErrShortWrite)
			}
			return ctx.Err()
		}()
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	}, nil
}

// MustCreate compiles the templates and panics on invalid configuration.
func (this *TemplatedHelpHandlerRequirement) MustCreate() HelpHandler {
	handler, err := this.Create()
	if err != nil {
		panic(err)
	}
	return handler
}

type helpTemplateSource struct {
	name  string
	value string
}

func createHelpTemplates(funcs template.FuncMap, sources []helpTemplateSource) (_ *template.Template, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("help template functions: %v", recovered)
		}
	}()
	templates := template.New("help").Funcs(funcs).Option("missingkey=error")
	for _, source := range sources {
		if _, err := templates.New(source.name).Parse(source.value); err != nil {
			return nil, fmt.Errorf("help template %s: %w", source.name, err)
		}
	}
	return templates, nil
}
