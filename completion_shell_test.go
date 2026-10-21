package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeShellCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the completion fixture and runs available native shells")
	}
	directory := t.TempDir()
	name := "cli4g-completion-test"
	binary := filepath.Join(directory, name)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := func(variant, publicCommand string) {
		flags := "-X main.variant=" + variant + " -X main.completionCommand=" + publicCommand + " -X main.completionRequestCommand=__suggest"
		command := exec.Command("go", "build", "-ldflags", flags, "-o", binary, "./internal/testdata/completion")
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	build("initial", "shell-setup")
	require.NoError(t, os.WriteFile(filepath.Join(directory, "config.yaml"), nil, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "space file.yaml"), nil, 0600))
	if runtime.GOOS != "windows" {
		require.NoError(t, os.WriteFile(filepath.Join(directory, "colon:file.yaml"), nil, 0600))
	}
	app := NewSimpleApplication(name).WithCompletion(true).
		WithCompletionCommand(NewCompletionCommand().WithName("shell-setup")).WithCompletionRequestCommandName("__suggest").MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	paths := make(map[CompletionShell]string)
	for _, shell := range []CompletionShell{CompletionBash, CompletionNushell} {
		path, err := app.InstallCompletion(nil, shell, directory)
		require.NoError(t, err)
		paths[shell] = path
	}
	assertBash := func(t *testing.T, line, current string, want []string) {
		t.Helper()
		if _, err := exec.LookPath("bash"); err != nil {
			t.Skip("Bash is not installed")
		}
		if err := exec.Command("bash", "-c", "((BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 4)))").Run(); err != nil {
			t.Skip("Bash 4.4 or newer is required")
		}
		command := exec.Command("bash", "--noprofile", "--norc", "-c", `
source "$1"
compopt() { :; }
COMP_LINE="$2"
COMP_POINT="$5"
[[ $(complete -p cli4g-completion-test) =~ -F\ ([^\ ]+) ]]
"${BASH_REMATCH[1]}" "$3" "$4" ""
if ((${#COMPREPLY[@]})); then printf '%s\0' "${COMPREPLY[@]}"; fi
`, "--", paths[CompletionBash], line, binary, current, strconv.Itoa(len(line)))
		command.Dir = directory
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		values := []string{}
		if len(output) > 0 {
			values = strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
		}
		assert.Equal(t, want, values)
	}
	assertNu := func(t *testing.T, line string, want []string) {
		t.Helper()
		if _, err := exec.LookPath("nu"); err != nil {
			t.Skip("Nushell is not installed")
		}
		command := exec.Command("nu", "-n", "-c", "source "+strconv.Quote(paths[CompletionNushell])+"; "+strconv.Quote(line)+" | commandline complete --detailed | to json --raw")
		command.Dir = directory
		command.Env = append(os.Environ(), "PATH="+directory+string(os.PathListSeparator)+os.Getenv("PATH"))
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		var candidates []codec.CompletionCandidate
		require.NoError(t, json.Unmarshal(output, &candidates), "%s", output)
		assert.Equal(t, want, completionTexts(codec.CompletionResult{Candidates: candidates}))
	}
	t.Run("bash", func(t *testing.T) {
		assertBash(t, name+" --format=ya", "ya", []string{"yaml"})
		assertBash(t, name+" --format ü", "ü", []string{"über"})
		assertBash(t, name+" --format 'two wo", "two wo", []string{"'two words'"})
		assertBash(t, name+" --context cluster-b --namespace cl", "cl", []string{"cluster-b-namespace"})
		assertBash(t, name+" get po", "po", []string{"pods"})
		assertBash(t, name+" --initial", "--initial", []string{"--initial"})
		assertBash(t, name+" --format nope", "nope", []string{})
		assertBash(t, name+" --input=config", "config", []string{"config.yaml"})
		assertBash(t, name+` --input="space fi`, "space fi", []string{`"space file.yaml"`})
		if runtime.GOOS != "windows" {
			assertBash(t, name+" --input=colon:fi", "fi", []string{"file.yaml"})
		}
	})
	t.Run("nushell", func(t *testing.T) {
		assertNu(t, name+" --format=ya", []string{"--format=yaml"})
		assertNu(t, name+" --format ü", []string{"über"})
		assertNu(t, name+" --format 'two wo", []string{`"two words"`})
		// The inserted representation must still be one literal native argument.
		command := exec.Command("nu", "-n", "-c", "^"+strconv.Quote(binary)+` __echo-args "two words"`)
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		var arguments []string
		require.NoError(t, json.Unmarshal(output, &arguments))
		assert.Equal(t, []string{"two words"}, arguments)
		assertNu(t, name+" --context cluster-b --namespace cl", []string{"cluster-b-namespace"})
		assertNu(t, name+" get po", []string{"pods"})
		assertNu(t, name+" --initial", []string{"--initial"})
		assertNu(t, name+" --input=config", []string{"--input=config.yaml"})
		assertNu(t, name+` --input="space fi`, []string{"--input=`space file.yaml`"})
	})
	// Reuse exactly the installed adapters against a replaced executable.
	build("updated", "shell-admin")
	t.Run("bash_after_binary_update", func(t *testing.T) {
		assertBash(t, name+" --up", "--up", []string{"--updated"})
	})
	t.Run("nushell_after_binary_update", func(t *testing.T) {
		assertNu(t, name+" --up", []string{"--updated"})
	})
}

func BenchmarkCompletionProcess(b *testing.B) {
	path := filepath.Join(b.TempDir(), "completion-test")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	output, err := exec.Command("go", "build", "-o", path, "./internal/testdata/completion").CombinedOutput()
	require.NoError(b, err, "%s", output)
	for b.Loop() {
		command := exec.Command(path, DefaultCompletionRequestCommandName, "1", "nu")
		command.Stdin = strings.NewReader(`{"words":["--format","j"],"word":1,"offset":1}`)
		if output, err := command.CombinedOutput(); err != nil {
			b.Fatalf("%v: %s", err, output)
		}
	}
}
