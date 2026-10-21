//go:build linux && shell_e2e

package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func testPowerShellStopParsingLiteral(t *testing.T) {
	environment := newShellEnvironment(t)
	environment.env = append(environment.env, "CLI4G_COMPLETION_LITERAL=--%", "CLI4G_MARKER_SENTINEL=must-not-expand")
	terminal := environment.start("powershell")
	name := filepath.Base(environment.program)
	for _, mode := range []string{"Standard", "Legacy"} {
		terminal.send("$PSNativeCommandArgumentPassing = '" + mode + "'\r")
		terminal.prompt()
		for _, test := range []struct {
			before, after string
			want          []string
		}{
			{" --namespace ", "", []string{"--namespace", "--%"}},
			{" --namespace ", " --context 'two words'", []string{"--namespace", "--%", "--context", "two words"}},
			{" --namespace ", " --context '%CLI4G_MARKER_SENTINEL%'", []string{"--namespace", "--%", "--context", "%CLI4G_MARKER_SENTINEL%"}},
			{" --namespace '", "' --context 'two words'", []string{"--namespace", "--%", "--context", "two words"}},
			{" --namespace=", " --context 'two words'", []string{"--namespace=--%", "--context", "two words"}},
		} {
			line := name + test.before + test.after
			completed := terminal.buffer(line, "\t", len(name+test.before))
			assertPowerShellCompletedArguments(t, environment, mode, completed, test.want...)
		}
		// Exercise actual Tab followed by Enter as well as executing the dumped
		// completion text. The fixture's echo mode never runs business handlers.
		terminal.send(name + " __echo-args --namespace \t --context 'two words'\r")
		matched := terminal.expect("(\\[\"--namespace\"[^\r\n]*\\])\r?\n")
		terminal.prompt()
		var args []string
		must(t, json.Unmarshal([]byte(matched[1]), &args))
		if want := []string{"--namespace", "--%", "--context", "two words"}; !slices.Equal(args, want) {
			t.Errorf("Tab/Enter (%s): got=%q want=%q", mode, args, want)
		}
	}
	environment.assertRequestsSkipInitialization()
}

func testPowerShellStopParsingDependency(t *testing.T) {
	environment := newShellEnvironment(t)
	terminal := environment.start("powershell")
	name := filepath.Base(environment.program)
	for _, mode := range []string{"Standard", "Legacy"} {
		terminal.send("$PSNativeCommandArgumentPassing = '" + mode + "'\r")
		terminal.prompt()
		for _, test := range []struct {
			before, after string
			want          []string
		}{
			{" --context @('--%') --namespace ", "", []string{"--context", "--%", "--namespace", "--%-namespace"}},
			{" --namespace ", " --context @('--%')", []string{"--namespace", "--%-namespace", "--context", "--%"}},
			{" --context '@(''--%'')' --namespace ", "", []string{"--context", "@('--%')", "--namespace", "@('--%')-namespace"}},
		} {
			line := name + test.before + test.after
			completed := terminal.buffer(line, "\t", len(name+test.before))
			assertPowerShellCompletedArguments(t, environment, mode, completed, test.want...)
		}
	}
	environment.assertRequestsSkipInitialization()
}

func assertPowerShellCompletedArguments(t *testing.T, environment *shellEnvironment, mode, completed string, want ...string) {
	t.Helper()
	name := filepath.Base(environment.program)
	echoed := strings.Replace(completed, name, name+" __echo-args", 1)
	command := "$PSNativeCommandArgumentPassing = '" + mode + "'\n" + echoed
	var args []string
	must(t, json.Unmarshal(environment.output("pwsh", "-NoLogo", "-NoProfile", "-Command", command), &args))
	if !slices.Equal(args, want) {
		t.Errorf("native arguments (%s): got=%q want=%q buffer=%q", mode, args, want, completed)
	}
}
