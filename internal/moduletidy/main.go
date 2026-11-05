// Command moduletidy keeps the root, benchmark, and example modules tidy.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/echocat/cli4g/internal/exampleutil"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, output, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("moduletidy", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	check := flags.Bool("check", false, "check module files without changing them")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(diagnostics, "usage: moduletidy [-check]")
		return 2
	}
	root, err := exampleutil.Root(".")
	if err == nil {
		err = tidy(root, *check, output, diagnostics)
	}
	if err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		if *check {
			_, _ = fmt.Fprintln(diagnostics, "Run mise run tidy to update all module files.")
		}
		return 1
	}
	return 0
}

func modules(root string) ([]string, error) {
	names, err := exampleutil.Names(root)
	if err != nil {
		return nil, err
	}
	result := []string{".", "_benchmarks"}
	for _, name := range names {
		result = append(result, filepath.Join("_examples", name))
	}
	// Validate the complete checkout before changing any module files.
	for _, name := range result {
		if _, err := os.Stat(filepath.Join(root, name, "go.mod")); err != nil {
			return nil, fmt.Errorf("module %s: %w", name, err)
		}
	}
	return result, nil
}

func tidy(root string, check bool, output, diagnostics io.Writer) error {
	directories, err := modules(root)
	if err != nil {
		return err
	}
	arguments := []string{"mod", "tidy"}
	verb := "Tidying"
	if check {
		arguments = append(arguments, "-diff")
		verb = "Checking"
	}
	// Nested modules replace cli4g with the root, so always tidy the root first.
	for _, directory := range directories {
		_, _ = fmt.Fprintf(output, "%s module %s\n", verb, filepath.ToSlash(directory))
		command := exec.Command("go", arguments...)
		command.Dir = filepath.Join(root, directory)
		command.Env = append(command.Environ(), "GOWORK=off")
		command.Stdout, command.Stderr = output, diagnostics
		if err := command.Run(); err != nil {
			return fmt.Errorf("go mod tidy in %s: %w", filepath.ToSlash(directory), err)
		}
	}
	return nil
}
