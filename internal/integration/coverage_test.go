package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/echocat/cli4g/internal/exampleutil"
	"github.com/echocat/cli4g/internal/testexec"
)

func exportExampleCoverage(t *testing.T, root, directory, profile string) {
	t.Helper()
	if !filepath.IsAbs(profile) {
		profile = filepath.Join(root, profile)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Log("no example coverage data: no instrumented examples ran")
		return
	}
	merged := t.TempDir()
	env := exampleutil.NativeEnv()
	// testing cancels t.Context before cleanup; exporting still has a bounded deadline.
	ctx := context.WithoutCancel(t.Context())
	runToolContext(t, ctx, root, env, "go", "tool", "covdata", "merge", "-pcombine", "-i="+directory, "-o="+merged)
	runToolContext(t, ctx, root, env, "go", "tool", "covdata", "textfmt", "-i="+merged, "-o="+profile)
	contents, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	// Nested example modules are not importable from the root module. Relative
	// source paths let go tool cover and coverage reporters locate their files.
	text := strings.ReplaceAll(string(contents), "\ngithub.com/echocat/cli4g/_examples/", "\n./_examples/")
	if err := os.WriteFile(profile, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	percent := runToolContext(t, ctx, root, env, "go", "tool", "covdata", "percent", "-i="+merged)
	// Keep nested binary statistics distinct from go test's own "coverage:" summary.
	t.Logf("example coverage profile: %s\n%s", profile, strings.ReplaceAll(percent.Stdout, "coverage:", "statements covered:"))
}

func TestExampleCoverageAcrossPrograms(t *testing.T) {
	directory := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":                  "module github.com/echocat/cli4g\n\ngo 1.27.0\n",
		"_examples/first/go.mod":  "module github.com/echocat/cli4g/_examples/first\n\ngo 1.27.0\n\nrequire github.com/echocat/cli4g v0.0.0\nreplace github.com/echocat/cli4g => ../..\n",
		"_examples/second/go.mod": "module github.com/echocat/cli4g/_examples/second\n\ngo 1.27.0\n\nrequire github.com/echocat/cli4g v0.0.0\nreplace github.com/echocat/cli4g => ../..\n",
		"shared/branches.go": `package shared
func Choose(success bool) int {
 if success {
  return 0
 }
 return 7
}
`,
		"_examples/first/main.go": `package main
import ("os"; "github.com/echocat/cli4g/shared")
func main() { os.Exit(shared.Choose(true)) }
`,
		"_examples/second/main.go": `package main
import ("os"; "github.com/echocat/cli4g/shared")
func main() { os.Exit(shared.Choose(false)) }
`,
	} {
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	coverageDirectory := t.TempDir()
	env := append(exampleutil.NativeEnv(), "GOCOVERDIR="+coverageDirectory)
	for index, name := range []string{"first", "second"} {
		binary := filepath.Join(t.TempDir(), "program.exe")
		runTool(t, filepath.Join(directory, "_examples", name), env, "go", "build", "-cover", "-covermode=atomic", "-coverpkg=.,github.com/echocat/cli4g/shared", "-o", binary, ".")
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		result, err := testexec.Run(ctx, binary, nil, testexec.Options{Dir: directory, Env: env})
		cancel()
		if err != nil || result.Code != index*7 || result.Stdout != "" || result.Stderr != "" {
			t.Fatalf("%s: %v, exit %d, stdout %q, stderr %q", name, err, result.Code, result.Stdout, result.Stderr)
		}
	}
	profile := filepath.Join(directory, "profile.cov")
	t.Cleanup(func() {
		exportExampleCoverage(t, directory, coverageDirectory, "profile.cov")
		contents, err := os.ReadFile(profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(contents), "mode: atomic\n") {
			t.Fatalf("unexpected profile mode: %s", contents)
		}
		sharedBlocks := make(map[string]uint64)
		mainPackages := make(map[string]bool)
		for _, line := range strings.Split(string(contents), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 3 {
				continue
			}
			count, err := strconv.ParseUint(fields[2], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(fields[0], "github.com/echocat/cli4g/shared/branches.go:") {
				if _, exists := sharedBlocks[fields[0]]; exists || count == 0 {
					t.Fatalf("shared block duplicated or uncovered: %s", line)
				}
				sharedBlocks[fields[0]] = count
			}
			for _, name := range []string{"first", "second"} {
				if strings.HasPrefix(fields[0], "./_examples/"+name+"/main.go:") && count > 0 {
					mainPackages[name] = true
				}
			}
		}
		if len(sharedBlocks) != 3 || !mainPackages["first"] || !mainPackages["second"] {
			t.Fatalf("coverage missing from a program or its shared dependency: %s", contents)
		}
		result := runToolContext(t, context.WithoutCancel(t.Context()), directory, exampleutil.NativeEnv(), "go", "tool", "cover", "-func="+profile)
		if !strings.Contains(result.Stdout, "100.0%") {
			t.Fatalf("go tool cover could not report full fixture coverage: %s", result.Stdout)
		}
	})
}
