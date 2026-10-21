//go:build linux && shell_e2e

package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func testNushellCursorContext(t *testing.T) {
	environment := newShellEnvironment(t)
	terminal := environment.start("nu")
	name := filepath.Base(environment.program)
	// Nu 0.114.1 passes only the command prefix to command-wide completers.
	// These version-specific cases document that native boundary, not a limit
	// of the Go completion API. Revisit them if Nu starts supplying later words.
	// Reuse one terminal so stale commandline snapshots cannot pass unnoticed.
	for _, test := range []struct {
		name, before, after, want string
		wantArgs                  []string
	}{
		{
			name: "dependency_before_cursor", before: " --context cluster-b --namespace cl", after: " --format yaml",
			want: " --context cluster-b --namespace cluster-b-namespace --format yaml",
		},
		{
			name: "inline_dependency", before: " --context=cluster-b --namespace=cl", after: " --format yaml",
			want: " --context=cluster-b --namespace=cluster-b-namespace --format yaml",
		},
		{
			name: "unicode_cursor", before: " --context über --namespace ü", after: " --format yaml",
			want:     " --context über --namespace über-namespace --format yaml",
			wantArgs: []string{"--context", "über", "--namespace", "über-namespace", "--format", "yaml"},
		},
		{
			name: "quoted_dependency", before: " --context 'cluster b' --namespace cl", after: " --format 'two words'",
			want:     ` --context 'cluster b' --namespace "cluster b-namespace" --format 'two words'`,
			wantArgs: []string{"--context", "cluster b", "--namespace", "cluster b-namespace", "--format", "two words"},
		},
		{
			name: "later_dependency_unavailable", before: " --namespace ", after: " --context cluster-b",
			want: " --namespace production --context cluster-b",
		},
		{
			name: "later_dependency_does_not_match_prefix", before: " --namespace cl", after: " --context cluster-b",
			want: " --namespace cl --context cluster-b",
		},
		{
			name: "prefix_uses_available_context", before: " --namespace pr", after: " --context cluster-b",
			want: " --namespace production --context cluster-b",
		},
		{
			name: "inline_later_dependency", before: " --namespace=", after: " --context cluster-b",
			want: " --namespace=production --context cluster-b",
		},
		{
			name: "unicode_later_dependency", before: " --namespace ü", after: " --context über",
			want: " --namespace ü --context über",
		},
		{
			name: "quoted_later_dependency", before: " --namespace cl", after: " --context 'cluster b'",
			want: " --namespace cl --context 'cluster b'",
		},
		{
			name: "fresh_dependency_in_same_session", before: " --context cluster-a --namespace cl", after: " --format json",
			want: " --context cluster-a --namespace cluster-a-namespace --format json",
		},
	} {
		line := name + test.before + test.after
		completed := terminal.buffer(line, "\t", len([]rune(name+test.before)))
		if strings.TrimRight(completed, " ") != name+test.want {
			t.Errorf("%s: got=%q want=%q", test.name, completed, name+test.want)
		}
		if test.wantArgs != nil {
			echoed := strings.Replace(completed, name, name+" __echo-args", 1)
			var args []string
			must(t, json.Unmarshal(environment.output("nu", "-n", "-c", echoed), &args))
			if !slices.Equal(args, test.wantArgs) {
				t.Errorf("%s native arguments: got=%q want=%q", test.name, args, test.wantArgs)
			}
		}
	}
	environment.assertRequestsSkipInitialization()
}
