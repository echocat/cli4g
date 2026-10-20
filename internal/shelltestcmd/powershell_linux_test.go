//go:build linux && shell_e2e

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerPowerShellCompletion(t *testing.T) {
	environment := newShellEnvironment(t)
	version := strings.TrimSpace(string(environment.output("pwsh", "-NoLogo", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()")))
	if version != "7.5.0" {
		t.Fatalf("unexpected PowerShell version: %q", version)
	}
	t.Logf("Testing PowerShell %s / PSReadLine in an actual terminal", version)
	profile := filepath.Join(environment.root, "config", "powershell", "profile.ps1")
	must(t, os.MkdirAll(filepath.Dir(profile), 0700))
	const userProfile = "$env:CLI4G_PROFILE_SENTINEL = 'kept'\n"
	must(t, os.WriteFile(profile, []byte(userProfile), 0600))
	must(t, os.WriteFile(filepath.Join(environment.root, "nope-file"), nil, 0600))
	terminal := environment.start("powershell")
	name := filepath.Base(environment.program)
	if got := strings.TrimSpace(string(environment.output("pwsh", "-NoLogo", "-Command", "$env:CLI4G_PROFILE_SENTINEL"))); got != "kept" {
		t.Fatalf("lost user profile content: %q", got)
	}
	for _, value := range []struct {
		line string
		want []string
	}{
		{name + " --format ü", []string{"über"}},
		{"Write-Output ignored | " + name + " --format=ya", []string{"--format=yaml"}},
	} {
		command := "$ErrorActionPreference = 'Stop'; $line = '" + strings.ReplaceAll(value.line, "'", "''") + "'; " +
			"$completion = [System.Management.Automation.CommandCompletion]::CompleteInput($line, $line.Length, $null); " +
			"ConvertTo-Json -InputObject @($completion.CompletionMatches | ForEach-Object CompletionText) -Compress"
		var matches []string
		must(t, json.Unmarshal(environment.output("pwsh", "-NoLogo", "-Command", command), &matches))
		if strings.Join(matches, "\x00") != strings.Join(value.want, "\x00") {
			t.Errorf("native completion API for %q: %q, want %q", value.line, matches, value.want)
		}
	}
	for _, value := range [][2]string{
		{" --format ya", " --format yaml"}, {" --format=ya", " --format=yaml"}, {" --format ü", " --format über"},
		{` --format "two wo`, ` --format 'two words'`}, {` --format="two wo`, ` '--format=two words'`},
		{" get po", " get pods"}, {" --context cluster-b --namespace cl", " --context cluster-b --namespace cluster-b-namespace"},
		{` --context 'two words' --namespace two`, ` --context 'two words' --namespace 'two words-namespace'`},
		{" --selector en", " --selector env="}, {" --ordered ", " --ordered zulu"},
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
	if got := terminal.buffer(name+" --ordered ", "\t\t", -1); got != name+" --ordered alpha" {
		t.Errorf("KeepOrder: %q", got)
	}
	must(t, os.Mkdir(filepath.Join(environment.root, "directory name"), 0700))
	must(t, os.WriteFile(filepath.Join(environment.root, "directory-file"), nil, 0600))
	if got := terminal.complete(name + " shell-setup install powershell directory"); got != name+" shell-setup install powershell './directory name/'" {
		t.Errorf("directory completion: %q", got)
	}
	for _, suffix := range []string{" --input=config.y", ` --input="space fi`, " --input=colon:fi"} {
		completed := terminal.complete(name + suffix)
		echoed := strings.Replace(completed, name, name+" __echo-args", 1)
		var argv []string
		must(t, json.Unmarshal(environment.output("pwsh", "-NoLogo", "-NoProfile", "-Command", echoed), &argv))
		want := "config.yaml"
		if strings.Contains(suffix, "space") {
			want = "space file.yaml"
		} else if strings.Contains(suffix, "colon") {
			want = "colon:file.yaml"
		}
		if len(argv) != 1 || !strings.HasPrefix(argv[0], "--input=") || filepath.Base(strings.TrimPrefix(argv[0], "--input=")) != want {
			t.Errorf("path completion %q changed arguments: %q (buffer %q)", suffix, argv, completed)
		}
	}
	// Literal candidate quoting is tested by native argument execution, including
	// PowerShell 7's legacy mode used to exercise Windows PowerShell 5.1 behavior.
	for _, mode := range []string{"Standard", "Legacy"} {
		terminal.send("$PSNativeCommandArgumentPassing = '" + mode + "'\r")
		terminal.prompt()
		for _, value := range [][2]string{
			{`'a"b'`, `a"b-namespace`}, {`'a\"b'`, `a\"b-namespace`},
			{`'a''b'`, `a'b-namespace`}, {`'a’’b'`, `a’b-namespace`},
			{`'$literal;value'`, `$literal;value-namespace`}, {`''`, `-namespace`},
		} {
			line := name + " --context " + value[0] + " --namespace "
			completed := terminal.complete(line)
			_, candidate, found := strings.Cut(completed, " --namespace ")
			if !found || candidate == "" {
				t.Fatalf("missing literal completion: %q", completed)
			}
			command := "$PSNativeCommandArgumentPassing = '" + mode + "'; " + name + " __echo-args " + candidate
			var argv []string
			must(t, json.Unmarshal(environment.output("pwsh", "-NoLogo", "-NoProfile", "-Command", command), &argv))
			if len(argv) != 1 || argv[0] != value[1] {
				t.Errorf("literal arguments (%s): %q, want %q (buffer %q)", mode, argv, value[1], completed)
			}
		}
	}
	if got := terminal.complete(name + " --ini"); got != name+" --initial" {
		t.Errorf("initial binary completion: %q", got)
	}
	environment.updateBinary()
	if got := terminal.complete(name + " --up"); got != name+" --updated" {
		t.Errorf("updated binary completion: %q", got)
	}
	environment.assertRequestsSkipInitialization()
	// Repeat installation with the current public command; no duplicate loader or
	// discarded user profile content, and no dependency on the old command name.
	before, err := os.ReadFile(profile)
	must(t, err)
	environment.output(environment.program, "shell-admin", "install", "powershell")
	after, err := os.ReadFile(profile)
	must(t, err)
	if string(before) != string(after) {
		t.Fatal("reinstallation changed the PowerShell profile")
	}
}
