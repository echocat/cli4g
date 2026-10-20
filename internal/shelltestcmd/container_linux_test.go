//go:build linux && shell_e2e

package main

import (
	"embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

//go:embed bash.sh config.nu powershell.ps1
var shellAssets embed.FS

type shellEnvironment struct {
	t       *testing.T
	root    string
	program string
	marker  string
	env     []string
}

func newShellEnvironment(t *testing.T) *shellEnvironment {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	must(t, os.MkdirAll(bin, 0700))
	program := filepath.Join(bin, "cli4g-completion-test")
	copyExecutable(t, "/opt/fixtures/fixture-initial", program)
	marker := filepath.Join(root, "initialized")
	env := append(os.Environ(),
		"HOME="+filepath.Join(root, "home"), "XDG_CONFIG_HOME="+filepath.Join(root, "config"),
		"XDG_DATA_HOME="+filepath.Join(root, "data"), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TERM=xterm-256color", "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "INPUTRC=/dev/null", "CLI4G_COMPLETION_INIT_MARKER="+marker)
	must(t, os.MkdirAll(filepath.Join(root, "home"), 0700))
	for _, name := range []string{"config.yaml", "space file.yaml", "colon:file.yaml"} {
		must(t, os.WriteFile(filepath.Join(root, name), nil, 0600))
	}
	return &shellEnvironment{t: t, root: root, program: program, marker: marker, env: env}
}

func copyExecutable(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	must(t, err)
	must(t, os.WriteFile(destination, data, 0755))
}

func (environment *shellEnvironment) output(name string, args ...string) []byte {
	environment.t.Helper()
	command := exec.Command(name, args...)
	command.Dir, command.Env = environment.root, environment.env
	output, err := command.CombinedOutput()
	if err != nil {
		environment.t.Fatalf("%s %q: %v\n%s", name, args, err, output)
	}
	return output
}

func (environment *shellEnvironment) assertRequestsSkipInitialization() {
	environment.t.Helper()
	data, err := os.ReadFile(environment.marker)
	must(environment.t, err)
	if string(data) != "initialize\n" {
		environment.t.Fatalf("completion ran normal initialization: %q", data)
	}
}

func (environment *shellEnvironment) start(shell string) *terminal {
	environment.t.Helper()
	environment.output(environment.program, "shell-setup", "install", shell)
	environment.assertRequestsSkipInitialization()
	if shell == "bash" {
		terminal := startTerminal(environment.t, shell, []string{"--noprofile", "--norc", "-i"}, environment.env, environment.root)
		setup, err := shellAssets.ReadFile("bash.sh")
		must(environment.t, err)
		terminal.send(string(setup))
		terminal.expect("\r?\n__CLI4G_READY__\r?\n")
		terminal.prompt()
		return terminal
	}
	if shell == "powershell" {
		setup, err := shellAssets.ReadFile("powershell.ps1")
		must(environment.t, err)
		terminal := startTerminal(environment.t, "pwsh", []string{"-NoLogo", "-NoExit", "-Command", string(setup)}, environment.env, environment.root)
		terminal.expect("__CLI4G_READY__\r?\n")
		terminal.prompt()
		return terminal
	}
	path := filepath.Join(environment.root, "config", "nushell", "config.nu")
	must(environment.t, os.MkdirAll(filepath.Dir(path), 0700))
	config, err := shellAssets.ReadFile("config.nu")
	must(environment.t, err)
	must(environment.t, os.WriteFile(path, config, 0600))
	terminal := startTerminal(environment.t, shell, []string{"--no-history", "--execute", `print "__CLI4G_READY__"`}, environment.env, environment.root)
	terminal.expect("\r?\n__CLI4G_READY__\r?\n|^__CLI4G_READY__\r?\n")
	terminal.prompt()
	return terminal
}

func (environment *shellEnvironment) updateBinary() {
	environment.t.Helper()
	replacement := filepath.Join(environment.root, "replacement")
	copyExecutable(environment.t, "/opt/fixtures/fixture-updated", replacement)
	must(environment.t, os.Rename(replacement, environment.program))
}

func TestContainerShellCompletion(t *testing.T) {
	bash, err := exec.Command("bash", "--version").CombinedOutput()
	must(t, err)
	nu, err := exec.Command("nu", "--version").CombinedOutput()
	must(t, err)
	if !strings.Contains(string(bash), "version 5.2.37") || strings.TrimSpace(string(nu)) != "0.114.1" {
		t.Fatalf("unexpected shell versions: %q, %q", bash, nu)
	}
	t.Log("Testing Bash 5.2.37 and Nushell 0.114.1 in actual terminals")
	t.Run("bash_readline", func(t *testing.T) {
		environment := newShellEnvironment(t)
		terminal := environment.start("bash")
		name := filepath.Base(environment.program)
		for _, value := range [][2]string{
			{" --format ya", " --format yaml "}, {" --format=ya", " --format=yaml "}, {" --format ü", " --format über "},
			{` --format "two wo`, ` --format "two words" `}, {" get po", " get pods "},
			{" --context cluster-b --namespace cl", " --context cluster-b --namespace cluster-b-namespace "},
			{" --input=config", " --input=config.yaml "}, {` --input="space fi`, ` --input="space file.yaml" `},
			{" --input=colon:fi", " --input=colon:file.yaml "}, {" --selector en", ` --selector env\=`},
		} {
			// Keep one terminal alive across all requests: this also checks that
			// the line editor and loaded completion options reset correctly.
			got := terminal.complete(name + value[0])
			if want := name + value[1]; got != want {
				t.Errorf("complete %q: got %q, want %q", value[0], got, want)
			}
		}
		completed := terminal.complete(name + " --selector en")
		echoed := strings.Replace(completed, name, name+" __echo-args", 1)
		var argv []string
		must(t, json.Unmarshal(environment.output("bash", "-c", echoed), &argv))
		if !slices.Equal(argv, []string{"--selector", "env="}) {
			t.Errorf("quoted NoSpace completion changed arguments: %q", argv)
		}
		environment.assertRequestsSkipInitialization()
	})
	t.Run("bash_order_cursor_and_binary_update", func(t *testing.T) {
		environment := newShellEnvironment(t)
		terminal := environment.start("bash")
		name := filepath.Base(environment.program)
		for _, value := range []struct{ suffix, keys, want string }{
			{" --ordered ", "\x18\x0e", " --ordered zulu "},
			{" --ordered ", "\x18\x0e\x18\x0e", " --ordered alpha "},
		} {
			if got := terminal.buffer(name+value.suffix, value.keys, -1); got != name+value.want {
				t.Errorf("KeepOrder: got %q, want %q", got, name+value.want)
			}
		}
		line := name + " --namespace cl --context cluster-b"
		want := name + " --namespace cluster-b-namespace --context cluster-b"
		if got := terminal.buffer(line, "\t", len(name+" --namespace cl")); got != want {
			t.Errorf("cursor completion: got %q, want %q", got, want)
		}
		if got := terminal.complete(name + " --ini"); got != name+" --initial " {
			t.Errorf("initial binary completion: %q", got)
		}
		environment.updateBinary()
		if got := terminal.complete(name + " --up"); got != name+" --updated " {
			t.Errorf("updated binary completion: %q", got)
		}
		environment.assertRequestsSkipInitialization()
	})
	t.Run("nushell_reedline_and_binary_update", func(t *testing.T) {
		environment := newShellEnvironment(t)
		terminal := environment.start("nu")
		name := filepath.Base(environment.program)
		for _, value := range [][2]string{
			{" --format ya", " --format yaml"}, {" --format=ya", " --format=yaml"}, {" --format ü", " --format über"},
			{` --format "two wo`, ` --format "two words"`}, {" get po", " get pods"},
			{" --context cluster-b --namespace cl", " --context cluster-b --namespace cluster-b-namespace"},
			{" --input=config.y", " --input=config.yaml"}, {` --input="space fi`, " --input=`space file.yaml`"},
		} {
			if got := strings.TrimRight(terminal.complete(name+value[0]), " "); got != name+value[1] {
				t.Errorf("complete %q: got %q, want %q", value[0], got, name+value[1])
			}
		}
		environment.updateBinary()
		if got := strings.TrimRight(terminal.complete(name+" --up"), " "); got != name+" --updated" {
			t.Errorf("updated binary completion: %q", got)
		}
		environment.assertRequestsSkipInitialization()
	})
}
