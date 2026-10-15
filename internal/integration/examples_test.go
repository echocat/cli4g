package integration

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/echocat/cli4g/internal/exampleutil"
	"github.com/echocat/cli4g/internal/testexec"
)

var exampleMode = flag.String("examples", "off", "example integration mode: off, code, smoke or check")
var exampleCoverProfile = flag.String("examples-coverprofile", "", "write coverage from example binaries to this file")

type smokeCase struct {
	name string
	args []string
	env  []string
	want expected
}

func TestExamples(t *testing.T) {
	if *exampleMode != "off" && *exampleMode != "code" && *exampleMode != "smoke" && *exampleMode != "check" {
		t.Fatalf("unknown examples mode %q", *exampleMode)
	}
	if *exampleMode == "off" {
		t.Skip("enable example checks with -examples=code, -examples=smoke or -examples=check")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := exampleutil.Root(cwd)
	if err != nil {
		t.Fatal(err)
	}
	names, err := exampleutil.Names(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := exampleCases()
	coverageDirectory := ""
	if *exampleCoverProfile != "" && (*exampleMode == "smoke" || *exampleMode == "check") {
		coverageDirectory = t.TempDir()
		t.Cleanup(func() {
			exportExampleCoverage(t, root, coverageDirectory, *exampleCoverProfile)
		})
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if len(cases[name]) == 0 {
				t.Fatal("example has no behavior case")
			}
			directory, err := exampleutil.Directory(root, name)
			if err != nil {
				t.Fatal(err)
			}
			env := exampleutil.NativeEnv()
			if *exampleMode == "code" || *exampleMode == "check" {
				t.Run("quality", func(t *testing.T) {
					formatted := runTool(t, directory, env, "gofmt", "-l", ".")
					if formatted.Stdout != "" {
						t.Fatalf("files need gofmt:\n%s", formatted.Stdout)
					}
					runTool(t, directory, env, "go", "mod", "tidy", "-diff")
					runTool(t, directory, env, "go", "vet", "./...")
					runTool(t, directory, env, "golangci-lint", "run", "--config", filepath.Join(root, ".golangci.yaml"))
				})
				if t.Failed() {
					return
				}
			}
			if *exampleMode == "code" {
				return
			}
			temporary := t.TempDir()
			binary := filepath.Join(temporary, name)
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			buildArguments := []string{"build", "-mod=readonly", "-o", binary}
			if coverageDirectory != "" {
				buildArguments = append(buildArguments, "-cover", "-covermode=atomic", "-coverpkg=.,github.com/echocat/cli4g/...")
				env = append(env, "GOCOVERDIR="+coverageDirectory)
			}
			runTool(t, directory, env, "go", append(buildArguments, ".")...)
			env = append(env, "CLI4G_PROFILE=development", "CLI4G_PORT=8080", "TMPDIR="+temporary, "TEMP="+temporary, "TMP="+temporary)
			t.Run("help", func(t *testing.T) {
				want := expected{stdout: text(""), stderrContains: "Usage:", help: boolean(true)}
				if name == "custom-help" {
					want = expected{stderr: text(""), help: boolean(true), helpStdout: true}
				}
				checkExample(t, directory, binary, env, []string{"--help"}, want)
			})
			for _, test := range cases[name] {
				t.Run(test.name, func(t *testing.T) {
					checkExample(t, directory, binary, append(append([]string(nil), env...), test.env...), test.args, test.want)
				})
			}
			if name == "cancellation" {
				t.Run("interrupt", func(t *testing.T) {
					if !testexec.CanInterrupt() {
						t.Skip("native targeted interrupt requires a supported platform and console")
					}
					ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
					defer cancel()
					result, err := testexec.Run(ctx, binary, []string{"--sleep", "1h"}, testexec.Options{
						Dir: directory, Env: env,
						ReadyStdout: regexp.MustCompile(`(?m)^Working\. Press Ctrl\+C to stop\.\r?$`),
						OnReady:     testexec.Interrupt,
					})
					if err != nil {
						t.Fatal(err)
					}
					assertExample(t, result, expected{code: 130, stdout: text("Working. Press Ctrl+C to stop.\n"), stderr: text("Error: context canceled\n"), help: boolean(false)})
				})
			}
		})
	}
}

func runTool(t *testing.T, directory string, env []string, command string, args ...string) testexec.Result {
	t.Helper()
	return runToolContext(t, t.Context(), directory, env, command, args...)
}

func runToolContext(t *testing.T, parent context.Context, directory string, env []string, command string, args ...string) testexec.Result {
	t.Helper()
	return runToolContextWithTimeout(t, parent, 2*time.Minute, directory, env, command, args...)
}

func runToolContextWithTimeout(t *testing.T, parent context.Context, timeout time.Duration, directory string, env []string, command string, args ...string) testexec.Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	result, err := testexec.Run(ctx, command, args, testexec.Options{Dir: directory, Env: env})
	if err != nil || result.Code != 0 {
		t.Fatalf("%s %q: %v, exit %d\nstdout:\n%s\nstderr:\n%s", command, args, err, result.Code, result.Stdout, result.Stderr)
	}
	return result
}

func checkExample(t *testing.T, directory, binary string, env, args []string, want expected) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	result, err := testexec.Run(ctx, binary, args, testexec.Options{Dir: directory, Env: env})
	if err != nil {
		t.Fatalf("%v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
	assertExample(t, result, want)
}

func assertExample(t *testing.T, result testexec.Result, want expected) {
	t.Helper()
	if err := checkOutput(result, want); err != nil {
		t.Fatalf("%v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
}
