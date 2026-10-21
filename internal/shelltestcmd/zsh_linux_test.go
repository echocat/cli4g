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

func testZshCompletion(t *testing.T) {
	environment := newShellEnvironment(t)
	profile := filepath.Join(environment.root, "home", ".zshrc")
	const userProfile = "export CLI4G_PROFILE_SENTINEL=kept\nzstyle ':completion:*' verbose yes\n"
	must(t, os.WriteFile(profile, []byte(userProfile), 0600))
	must(t, os.WriteFile(filepath.Join(environment.root, "nope-file"), nil, 0600))
	terminal := environment.start("zsh")
	name := filepath.Base(environment.program)
	for _, value := range [][2]string{
		{" --format ya", " --format yaml "}, {" --format=ya", " --format=yaml "}, {" --format ü", " --format über "},
		{` --format "two wo`, ` --format "two words" `}, {" get po", " get pods "},
		{" --context cluster-b --namespace cl", " --context cluster-b --namespace cluster-b-namespace "},
		{" --input=config.y", " --input=config.yaml "}, {` --input="space fi`, ` --input=space\ file.yaml `},
		{" --input=colon:fi", " --input=colon:file.yaml "}, {" --selector en", " --selector env="},
		{" --format nope", " --format nope"},
	} {
		if got := terminal.complete(name + value[0]); got != name+value[1] {
			t.Errorf("complete %q: got %q, want %q", value[0], got, name+value[1])
		}
	}
	for _, value := range []struct{ keys, want string }{
		{"\x18\x0e", "zulu"}, {"\x18\x0e\x18\x0e", "alpha"},
	} {
		if got := terminal.buffer(name+" --ordered ", value.keys, -1); got != name+" --ordered "+value.want {
			t.Errorf("KeepOrder: %q", got)
		}
	}
	line := name + " --namespace cl --context cluster-b"
	if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != name+" --namespace cluster-b-namespace --context cluster-b" {
		t.Errorf("cursor completion: %q", got)
	}
	line = name + " --namespace cluster-a-namespace --context cluster-b"
	if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != name+" --namespace cluster-b-namespace --context cluster-b" {
		t.Errorf("mid-word cursor completion: %q", got)
	}
	line = name + " --namespace ü --context über"
	if got := terminal.buffer(line, "\t", len([]rune(name+" --namespace ü"))); got != name+" --namespace über-namespace --context über" {
		t.Errorf("Unicode cursor completion: %q", got)
	}
	must(t, os.Mkdir(filepath.Join(environment.root, "directory name"), 0700))
	must(t, os.WriteFile(filepath.Join(environment.root, "directory-file"), nil, 0600))
	completed := terminal.complete(name + " shell-setup install zsh directory")
	var arguments []string
	must(t, json.Unmarshal(environment.output("zsh", "-f", "-c", strings.Replace(completed, name, name+" __echo-args", 1)), &arguments))
	if !slices.Equal(arguments, []string{"shell-setup", "install", "zsh", "directory name/"}) {
		t.Errorf("directory completion: %q (buffer %q)", arguments, completed)
	}
	for _, value := range [][2]string{{`'two words'`, "two words-namespace"}, {`'a'\''b'`, "a'b-namespace"}, {`'$(not-executed)'`, "$(not-executed)-namespace"}, {`''`, "-namespace"}} {
		completed := terminal.complete(name + " --context " + value[0] + " --namespace ")
		var argv []string
		must(t, json.Unmarshal(environment.output("zsh", "-f", "-c", strings.Replace(completed, name, name+" __echo-args", 1)), &argv))
		if !slices.Equal(argv, []string{"--context", strings.TrimSuffix(value[1], "-namespace"), "--namespace", value[1]}) {
			t.Errorf("literal arguments: %q (buffer %q)", argv, completed)
		}
	}
	if got := terminal.complete(name + " --ini"); got != name+" --initial " {
		t.Errorf("initial binary: %q", got)
	}
	environment.updateBinary()
	if got := terminal.complete(name + " --up"); got != name+" --updated " {
		t.Errorf("updated binary: %q", got)
	}
	environment.assertRequestsSkipInitialization()
	before, err := os.ReadFile(profile)
	must(t, err)
	environment.output(environment.program, "shell-admin", "install", "zsh")
	after, err := os.ReadFile(profile)
	must(t, err)
	if string(before) != string(after) || !strings.HasPrefix(string(after), userProfile) {
		t.Fatal("reinstallation changed user startup content or duplicated the loader")
	}
}

func testZshCustomStartup(t *testing.T) {
	environment := newShellEnvironment(t)
	startup := filepath.Join(environment.root, "custom zsh's home")
	must(t, os.MkdirAll(startup, 0700))
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	// ZDOTDIR is deliberately not exported. Installation must discover .zshenv's
	// setting rather than assuming the child application's environment has it.
	must(t, os.WriteFile(filepath.Join(environment.root, "home", ".zshenv"), []byte("ZDOTDIR="+quote(startup)+"\n"), 0600))
	profile := filepath.Join(startup, ".zshrc")
	const existing = "autoload -Uz compinit\ncompinit\nzstyle ':completion:*' verbose yes\n"
	must(t, os.WriteFile(profile, []byte(existing), 0600))
	terminal := environment.start("zsh")
	name := filepath.Base(environment.program)
	if got := terminal.complete(name + " --format ya"); got != name+" --format yaml " {
		t.Errorf("custom ZDOTDIR completion: %q", got)
	}
	// A packager can also source the same generated adapter after compinit.
	path := filepath.Join(environment.root, "data", "zsh", "site-functions", "_"+name)
	registration := environment.output("zsh", "-f", "-c", "autoload -Uz compinit; compinit; source "+quote(path)+"; print -r -- $_comps["+name+"]")
	if !strings.HasPrefix(string(registration), "_cli4g_") {
		t.Errorf("sourced adapter was not registered: %q", registration)
	}
	environment.assertRequestsSkipInitialization()
	before, err := os.ReadFile(profile)
	must(t, err)
	environment.output(environment.program, "shell-setup", "install", "zsh")
	after, err := os.ReadFile(profile)
	must(t, err)
	if string(before) != string(after) || !strings.HasPrefix(string(after), existing) {
		t.Fatal("custom ZDOTDIR setup was not preserved or idempotent")
	}
}
