//go:build linux && shell_e2e

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Exercise the actual Readline buffer and native argument passing, not only
// COMPREPLY: Readline's filename quoting alone does not make values literal.
func testBashLiteralCompletion(t *testing.T) {
	for _, candidate := range []string{
		"", "plain", "two words", "a'b", `a"b`, `a\b`, "über😀", "a\nb", "a\tb",
		"$HOME", "${HOME}", "$(printf literal)", "`printf literal`", "a;literal",
		"!literal", "*?[literal]", "~", "#literal", "(literal)", "trailing\\", "ends'", `ends"`,
	} {
		t.Run(candidate, func(t *testing.T) {
			environment := newShellEnvironment(t)
			environment.env = append(environment.env, "CLI4G_COMPLETION_LITERAL="+candidate)
			terminal := environment.start("bash")
			name := filepath.Base(environment.program)
			for _, prefix := range []string{"", "'", `"`, "''", `""`} {
				// The fixture's echo mode executes only after completion. The
				// request still goes through the real adapter and CLI scanner.
				completed := terminal.complete(name + " __echo-args --namespace " + prefix)
				assertBashCompletedArguments(t, terminal, completed, "--namespace", candidate)
				completed = terminal.complete(name + " __echo-args --namespace=" + prefix)
				assertBashCompletedArguments(t, terminal, completed, "--namespace="+candidate)
			}
			for _, quote := range []string{"", "'", `"`} {
				before := name + " __echo-args --namespace " + quote
				line := before + quote + " --context unchanged"
				completed := terminal.buffer(line, "\t", len([]rune(before)))
				assertBashCompletedArguments(t, terminal, completed, "--namespace", candidate, "--context", "unchanged")
			}
			line := name + " __echo-args --namespace $'"
			if completed := terminal.complete(line); completed != line {
				t.Errorf("unsupported ANSI-C quote must not receive simple-quote syntax: %q", completed)
			}
			// A completed ANSI-C quote in an earlier word must not disable
			// completion of the current, normally quoted argument.
			completed := terminal.complete(name + " __echo-args --context $'earlier' --namespace ")
			assertBashCompletedArguments(t, terminal, completed, "--context", "earlier", "--namespace", candidate)
			environment.assertRequestsSkipInitialization()
		})
	}
}

func testBashLiteralPaths(t *testing.T) {
	for _, name := range []string{"literal-$HOME", "literal-`name`", "literal-a'b", `literal-a"b`, `literal-a\b`, "literal-two words", "literal-!name"} {
		t.Run(name, func(t *testing.T) {
			environment := newShellEnvironment(t)
			must(t, os.WriteFile(filepath.Join(environment.root, name), nil, 0600))
			terminal := environment.start("bash")
			program := filepath.Base(environment.program)
			for _, quote := range []string{"", "'", `"`} {
				for _, flag := range []string{"--input ", "--input="} {
					completed := terminal.complete(program + " __echo-args " + flag + quote + "literal-")
					want := []string{"--input", name}
					if flag == "--input=" {
						want = []string{"--input=" + name}
					}
					assertBashCompletedArguments(t, terminal, completed, want...)
				}
			}
			environment.assertRequestsSkipInitialization()
		})
	}
	t.Run("directory_and_child", func(t *testing.T) {
		environment := newShellEnvironment(t)
		const directory = "literal directory"
		const child = "child file.yaml"
		must(t, os.Mkdir(filepath.Join(environment.root, directory), 0700))
		must(t, os.WriteFile(filepath.Join(environment.root, directory, child), nil, 0600))
		terminal := environment.start("bash")
		program := filepath.Base(environment.program)
		for _, quote := range []string{"", "'", `"`} {
			completed := terminal.complete(program + " __echo-args --input " + quote + "literal")
			if strings.HasSuffix(completed, " ") {
				t.Errorf("directory completion must not append a space: %q", completed)
			}
			assertBashCompletedArguments(t, terminal, completed, "--input", directory+"/")
			completed = terminal.complete(completed + "chil")
			assertBashCompletedArguments(t, terminal, completed, "--input", directory+"/"+child)
		}
		environment.assertRequestsSkipInitialization()
	})
}

func assertBashCompletedArguments(t *testing.T, terminal *terminal, completed string, want ...string) {
	t.Helper()
	terminal.send("\r")
	matched := terminal.expect("\r?\n(\\[.*?\\])\r?\n")
	terminal.prompt()
	var argv []string
	must(t, json.Unmarshal([]byte(matched[1]), &argv))
	if !slices.Equal(argv, want) {
		t.Errorf("literal completion changed arguments: want=%q got=%q buffer=%q", want, argv, completed)
	}
}
