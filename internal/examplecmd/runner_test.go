package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/echocat/cli4g/internal/exampleutil"
)

func TestTemporaryRunners(t *testing.T) {
	directory := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod": "module runnerfixture\n\ngo 1.27.0\n",
		"main.go": `package main
import ("encoding/json"; "os"; "strconv")
func main() {
 cwd, _ := os.Getwd()
 binary, _ := os.Executable()
 _ = json.NewEncoder(os.Stdout).Encode(map[string]any{"directory": cwd, "binary": binary, "args": os.Args[1:]})
 _, _ = os.Stderr.WriteString("fixture diagnostics\n")
 code, _ := strconv.Atoi(os.Args[1])
 os.Exit(code)
}
`,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, runner := range []string{"examplecmd", "mise"} {
		t.Run(runner, func(t *testing.T) {
			root := ""
			if runner == "mise" {
				if _, err := exec.LookPath("mise"); err != nil {
					t.Skip("mise is not installed")
				}
				var err error
				root, err = exampleutil.Root(".")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(root, ".git")); errors.Is(err, os.ErrNotExist) {
					t.Skip("mise task integration requires a Git checkout")
				} else if err != nil {
					t.Fatal(err)
				}
			}
			execute := func(pkg string, args []string, output, diagnostics *bytes.Buffer) int {
				if runner == "examplecmd" {
					return buildAndRun(directory, pkg, args, nil, output, diagnostics)
				}
				command := exec.Command("mise", append([]string{"run", "go:run", directory, pkg}, args...)...)
				command.Dir = root
				command.Env = append(command.Environ(), exampleutil.NativeEnv()...)
				command.Stdout, command.Stderr = output, diagnostics
				if err := command.Run(); err != nil {
					if exit, ok := errors.AsType[*exec.ExitError](err); ok {
						return nativeExitCode(exit)
					}
					t.Fatal(err)
				}
				return 0
			}
			for _, code := range []int{0, 2, 64, 70, 126, 127, 130} {
				t.Run(strconv.Itoa(code), func(t *testing.T) {
					arguments := []string{strconv.Itoa(code), "", "two words", `quotes'" and $dollar; & back\slash`, "--help"}
					var output, diagnostics bytes.Buffer
					if got := execute(".", arguments, &output, &diagnostics); got != code {
						t.Fatalf("exit %d, want %d: %s", got, code, diagnostics.String())
					}
					var result struct {
						Directory string   `json:"directory"`
						Binary    string   `json:"binary"`
						Args      []string `json:"args"`
					}
					if err := json.Unmarshal(output.Bytes(), &result); err != nil {
						t.Fatalf("decode output %q: %v", output.String(), err)
					}
					if !reflect.DeepEqual(result.Args, arguments) {
						t.Fatalf("args: got %#v, want %#v", result.Args, arguments)
					}
					gotDirectory, err := filepath.EvalSymlinks(result.Directory)
					if err != nil {
						t.Fatal(err)
					}
					wantDirectory, err := filepath.EvalSymlinks(directory)
					if err != nil {
						t.Fatal(err)
					}
					if gotDirectory != wantDirectory {
						t.Fatalf("directory: got %q, want %q", gotDirectory, wantDirectory)
					}
					if result.Binary == "" {
						t.Fatal("missing executable path")
					}
					if _, err := os.Stat(filepath.Dir(result.Binary)); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("temporary binary directory remains: %q: %v", result.Binary, err)
					}
					if !strings.Contains(diagnostics.String(), "fixture diagnostics\n") || strings.Contains(diagnostics.String(), "exit status") {
						t.Fatalf("unexpected diagnostics: %q", diagnostics.String())
					}
				})
			}
			t.Run("build failure", func(t *testing.T) {
				var output, diagnostics bytes.Buffer
				if code := execute("./missing", nil, &output, &diagnostics); code != 6 {
					t.Fatalf("build failure exit %d: %s", code, diagnostics.String())
				}
			})
		})
	}
}
