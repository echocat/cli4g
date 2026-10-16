package integration

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/echocat/cli4g/internal/testexec"
)

type expected struct {
	code              int
	stdout            *string
	stderr            *string
	stderrContains    string
	stderrNotContains []string
	help              *bool
	helpStdout        bool
}

func text(value string) *string { return &value }
func boolean(value bool) *bool  { return &value }

var helpHeader = regexp.MustCompile(`(?m)^Usage:[ \t]*$|^Run: [^\r\n]+$`)
var errorHeader = regexp.MustCompile(`(?m)^Error: `)

func checkOutput(result testexec.Result, want expected) error {
	if result.Code != want.code {
		return fmt.Errorf("exit code %d; want %d", result.Code, want.code)
	}
	if want.stdout != nil && result.Stdout != *want.stdout {
		return fmt.Errorf("stdout %q; want %q", result.Stdout, *want.stdout)
	}
	if want.stderr != nil {
		if result.Stderr != *want.stderr {
			return fmt.Errorf("stderr %q; want %q", result.Stderr, *want.stderr)
		}
	} else if want.stderrContains != "" {
		if !strings.Contains(result.Stderr, want.stderrContains) {
			return fmt.Errorf("stderr is missing %q", want.stderrContains)
		}
	} else if want.code == 0 && result.Stderr != "" {
		return fmt.Errorf("unexpected stderr %q", result.Stderr)
	}
	if want.code != 0 && !errorHeader.MatchString(result.Stderr) {
		return fmt.Errorf("missing error diagnostic on stderr")
	}
	for _, forbidden := range want.stderrNotContains {
		if strings.Contains(result.Stderr, forbidden) {
			return fmt.Errorf("stderr unexpectedly contains %q", forbidden)
		}
	}
	if want.help != nil {
		selected, other := result.Stderr, result.Stdout
		if want.helpStdout {
			selected, other = other, selected
		}
		if helpHeader.MatchString(selected) != *want.help || helpHeader.MatchString(other) {
			return fmt.Errorf("help missing, unexpected or on the wrong output channel")
		}
	}
	if strings.Contains(result.Stderr, "panic:") || strings.Contains(result.Stderr, "DATA RACE") {
		return fmt.Errorf("panic or race diagnostic on stderr")
	}
	return nil
}

func TestOutputRejectsFalsePositives(t *testing.T) {
	for _, test := range []struct {
		name string
		got  testexec.Result
		want expected
	}{
		{"missing error", testexec.Result{Code: 70}, expected{code: 70, help: boolean(false)}},
		{"unexpected stderr", testexec.Result{Stdout: "ok\n", Stderr: "unexpected\n"}, expected{stdout: text("ok\n")}},
		{"wrong help channel", testexec.Result{Stderr: "Run: app\n"}, expected{help: boolean(true), helpStdout: true}},
		{"help marker in error", testexec.Result{Code: 2, Stderr: "Error: failed to render Usage: template\n"}, expected{code: 2, help: boolean(true)}},
		{"echoed input", testexec.Result{Code: 2, Stderr: "Error: private-value\n"}, expected{code: 2, stderrNotContains: []string{"private-value"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := checkOutput(test.got, test.want); err == nil {
				t.Fatal("invalid output was accepted")
			}
		})
	}
}
