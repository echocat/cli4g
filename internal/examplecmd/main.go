// Command examplecmd implements mise's example run/list tasks using only Go.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/echocat/cli4g/internal/exampleutil"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, input io.Reader, output, diagnostics io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(diagnostics, "usage: examplecmd run [name] [arguments...] | list [-F text|json]")
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 1
	}
	root, err := exampleutil.Root(cwd)
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 1
	}
	switch args[0] {
	case "list":
		return list(root, args[1:], output, diagnostics)
	case "run":
		name := "simple"
		arguments := args[1:]
		if len(arguments) > 0 {
			name, arguments = arguments[0], arguments[1:]
		}
		directory, err := exampleutil.Directory(root, name)
		if err != nil {
			_, _ = fmt.Fprintln(diagnostics, "error:", err)
			return 2
		}
		return buildAndRun(directory, ".", arguments, input, output, diagnostics)
	default:
		_, _ = fmt.Fprintln(diagnostics, "unknown example command:", args[0])
		return 2
	}
}

func buildAndRun(directory, packageName string, args []string, input io.Reader, output, diagnostics io.Writer) int {
	temporary, err := os.MkdirTemp("", "cli4g-example-")
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 6
	}
	defer func() {
		if err := os.RemoveAll(temporary); err != nil {
			_, _ = fmt.Fprintln(diagnostics, "temporary binary cleanup:", err)
		}
	}()
	absoluteTemporary, err := filepath.Abs(temporary)
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 6
	}
	temporary = absoluteTemporary
	binary := filepath.Join(temporary, "program")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	// Keep this wrapper alive until foreground children finish handling Ctrl+C.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	build := exec.Command("go", "build", "-mod=readonly", "-o", binary, packageName)
	build.Dir = directory
	build.Env = append(build.Environ(), exampleutil.NativeEnv()...)
	build.Stdout, build.Stderr = output, diagnostics
	if err := build.Run(); err != nil {
		_, _ = fmt.Fprintln(diagnostics, "build example:", err)
		return 6
	}
	command := exec.Command(binary, args...)
	command.Dir = directory
	command.Env = append(command.Environ(), exampleutil.NativeEnv()...)
	command.Stdin, command.Stdout, command.Stderr = input, output, diagnostics
	if err := command.Start(); err != nil {
		_, _ = fmt.Fprintln(diagnostics, "start example:", err)
		return 6
	}
	if err := command.Wait(); err != nil {
		if exit, ok := errors.AsType[*exec.ExitError](err); ok {
			return nativeExitCode(exit)
		}
		_, _ = fmt.Fprintln(diagnostics, "wait for example:", err)
		return 6
	}
	return 0
}

func list(root string, args []string, output, diagnostics io.Writer) int {
	format := os.Getenv("usage_format")
	if format == "" {
		format = "text"
	}
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.StringVar(&format, "format", format, "output format: text or json")
	flags.StringVar(&format, "F", format, "output format: text or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || format != "text" && format != "json" {
		_, _ = fmt.Fprintln(diagnostics, "format must be text or json")
		return 2
	}
	names, err := exampleutil.Names(root)
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 1
	}
	if format == "json" {
		err = json.NewEncoder(output).Encode(names)
	} else {
		_, err = fmt.Fprintln(output, strings.Join(names, "\n"))
	}
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}
