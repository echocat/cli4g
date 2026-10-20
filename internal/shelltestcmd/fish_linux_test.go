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

func testFishCompletion(t *testing.T) {
	environment := newShellEnvironment(t)
	profile := filepath.Join(environment.root, "config", "fish", "config.fish")
	must(t, os.MkdirAll(filepath.Dir(profile), 0700))
	const userProfile = "set -gx CLI4G_PROFILE_SENTINEL kept\n"
	must(t, os.WriteFile(profile, []byte(userProfile), 0600))
	must(t, os.WriteFile(filepath.Join(environment.root, "nope-file"), nil, 0600))
	terminal := environment.start("fish")
	name := filepath.Base(environment.program)
	quote := func(value string) string {
		return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(value) + "'"
	}
	for _, value := range [][2]string{
		{" --format ya", " --format yaml "}, {" --format=ya", " --format=yaml "}, {" --format ü", " --format über "},
		{` --format "two wo`, ` --format "two words" `}, {" get po", " get pods "},
		{" --context cluster-b --namespace cl", " --context cluster-b --namespace cluster-b-namespace "},
		{" --input=config.y", " --input=config.yaml "}, {` --input="space fi`, ` --input="space file.yaml" `},
		{" --input=colon:fi", " --input=colon:file.yaml "}, {" --selector en", " --selector env="},
		{" --format nope", " --format nope"},
	} {
		if got := terminal.complete(name + value[0]); got != name+value[1] {
			t.Errorf("complete %q: got %q, want %q", value[0], got, name+value[1])
		}
	}
	line := name + " --namespace cl --context cluster-b"
	if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != name+" --namespace cluster-b-namespace --context cluster-b" {
		t.Errorf("cursor completion: %q", got)
	}
	line = name + " --namespace ü --context über"
	if got := terminal.buffer(line, "\t", len([]rune(name+" --namespace ü"))); got != name+" --namespace über-namespace --context über" {
		t.Errorf("Unicode cursor completion: %q", got)
	}
	line = name + " --namespace cluster-a-namespace --context cluster-b"
	if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != line {
		t.Errorf("Fish must retain a conflicting word suffix: %q", got)
	}
	line = name + " --namespace cl-namespace --context cluster-b"
	if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != name+" --namespace cluster-b-namespace --context cluster-b" {
		t.Errorf("compatible mid-word suffix completion: %q", got)
	}
	if got := terminal.complete("command " + name + " --format ya"); got != "command "+name+" --format yaml " {
		t.Errorf("command decorator: %q", got)
	}
	if got := terminal.complete("CONTEXT=ignored " + name + " --format ya"); got != "CONTEXT=ignored "+name+" --format yaml " {
		t.Errorf("environment prefix: %q", got)
	}
	for _, value := range []struct{ keys, want string }{
		{"\t\t", "zulu "}, {"\t\t\t", "alpha "},
	} {
		if got := terminal.buffer(name+" --ordered ", value.keys, -1); got != name+" --ordered "+value.want {
			t.Errorf("KeepOrder: %q", got)
		}
	}
	for _, value := range [][2]string{
		{`'two words'`, "two words"}, {`'a\'b'`, "a'b"}, {`'$(not-executed)'`, "$(not-executed)"},
		{`''`, ""}, {`a\nb`, "a\nb"}, {`\u00fcber`, "über"},
	} {
		completed := terminal.complete(name + " --context " + value[0] + " --namespace ")
		var argv []string
		must(t, json.Unmarshal(environment.output("fish", "--no-config", "-c", strings.Replace(completed, name, name+" __echo-args", 1)), &argv))
		if !slices.Equal(argv, []string{"--context", value[1], "--namespace", value[1] + "-namespace"}) {
			t.Errorf("literal arguments: %q (buffer %q)", argv, completed)
		}
	}
	substitutionMarker := filepath.Join(environment.root, "substitution-was-executed")
	expression := "(touch " + quote(substitutionMarker) + ")"
	substitution := terminal.complete(name + " --context " + expression + " --namespace ")
	if _, err := os.Stat(substitutionMarker); !os.IsNotExist(err) {
		t.Fatalf("completion evaluated a command substitution: %v", err)
	}
	_, candidate, found := strings.Cut(substitution, " --namespace ")
	if !found || strings.TrimSpace(candidate) == "" {
		t.Fatalf("missing literal substitution completion: %q", substitution)
	}
	var literal []string
	must(t, json.Unmarshal(environment.output("fish", "--no-config", "-c", name+" __echo-args "+candidate), &literal))
	if !slices.Equal(literal, []string{expression + "-namespace"}) {
		t.Errorf("literal substitution candidate: %q", literal)
	}
	must(t, os.Mkdir(filepath.Join(environment.root, "directory name"), 0700))
	must(t, os.WriteFile(filepath.Join(environment.root, "directory-file"), nil, 0600))
	completed := terminal.complete(name + " shell-setup install fish directory")
	var args []string
	must(t, json.Unmarshal(environment.output("fish", "--no-config", "-c", strings.Replace(completed, name, name+" __echo-args", 1)), &args))
	if len(args) != 4 || filepath.Base(strings.TrimSuffix(args[3], "/")) != "directory name" {
		t.Errorf("directory completion: %q (buffer %q)", args, completed)
	}
	// Sourcing works outside autoload too, and preserves native ordering and
	// descriptions in Fish's own completion API.
	path := filepath.Join(environment.root, "config", "fish", "completions", name+".fish")
	ordered := environment.output("fish", "--no-config", "-c", "source "+quote(path)+"; complete -C "+quote(name+" --ordered "))
	var values []string
	for _, row := range strings.Split(strings.TrimSuffix(string(ordered), "\n"), "\n") {
		value, _, _ := strings.Cut(row, "\t")
		values = append(values, value)
	}
	if !slices.Equal(values, []string{"zulu", "alpha"}) {
		t.Errorf("native ordering: %q", ordered)
	}
	described := environment.output("fish", "--no-config", "-c", "source "+quote(path)+"; complete -C "+quote(name+" --context cluster-b --namespace cl"))
	if !strings.Contains(string(described), "cluster-b-namespace\tNamespace in the selected cluster.") {
		t.Errorf("native description: %q", described)
	}
	if got := terminal.complete(name + " --ini"); got != name+" --initial " {
		t.Errorf("initial binary completion: %q", got)
	}
	environment.updateBinary()
	if got := terminal.complete(name + " --up"); got != name+" --updated " {
		t.Errorf("updated binary completion: %q", got)
	}
	environment.assertRequestsSkipInitialization()
	before, err := os.ReadFile(profile)
	must(t, err)
	environment.output(environment.program, "shell-admin", "install", "fish")
	after, err := os.ReadFile(profile)
	must(t, err)
	if string(before) != string(after) || !strings.HasPrefix(string(after), userProfile) {
		t.Fatal("installation changed the Fish config")
	}
	for _, diagnostic := range []string{"Invalid index value", "Expansion error", "Unknown command"} {
		if strings.Contains(string(terminal.transcript), diagnostic) {
			t.Errorf("Fish emitted a diagnostic: %q", diagnostic)
		}
	}
}
