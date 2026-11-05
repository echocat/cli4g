package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestModules(t *testing.T) {
	root := fixture(t)
	got, err := modules(root)
	want := []string{".", "_benchmarks", filepath.Join("_examples", "alpha"), filepath.Join("_examples", "simple")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("modules = %v, %v; want %v", got, err, want)
	}
}

func TestMissingModuleDoesNotChangeFiles(t *testing.T) {
	for _, name := range []string{"go.mod", "_benchmarks/go.mod", "_examples/alpha/go.mod"} {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
				t.Fatal(err)
			}
			before := moduleFiles(t, root)
			var output, diagnostics bytes.Buffer
			if err := tidy(root, false, &output, &diagnostics); err == nil {
				t.Fatal("tidy accepted a checkout with a missing module")
			}
			if got := moduleFiles(t, root); !reflect.DeepEqual(got, before) {
				t.Fatal("tidy modified files before validating the checkout")
			}
		})
	}
}

func TestTidyAndCheck(t *testing.T) {
	root := fixture(t)
	// This test needs no network and must ignore the caller's workspace.
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", filepath.Join(root, "missing.work"))
	t.Setenv("GOOS", runtime.GOOS)
	t.Setenv("GOARCH", runtime.GOARCH)
	before := moduleFiles(t, root)
	var output, diagnostics bytes.Buffer
	if err := tidy(root, true, &output, &diagnostics); err == nil {
		t.Fatal("check accepted untidy module files")
	}
	if !strings.Contains(output.String(), "diff current/go.mod tidy/go.mod") {
		t.Fatalf("check did not report the diff:\n%s\n%s", &output, &diagnostics)
	}
	if got := moduleFiles(t, root); !reflect.DeepEqual(got, before) {
		t.Fatal("check modified module files")
	}
	output.Reset()
	diagnostics.Reset()
	if err := tidy(root, false, &output, &diagnostics); err != nil {
		t.Fatalf("tidy: %v\n%s\n%s", err, &output, &diagnostics)
	}
	wantOutput := "Tidying module .\nTidying module _benchmarks\nTidying module _examples/alpha\nTidying module _examples/simple\n"
	if output.String() != wantOutput {
		t.Fatalf("unexpected module order: %q", output.String())
	}
	after := moduleFiles(t, root)
	if reflect.DeepEqual(after, before) {
		t.Fatal("tidy did not update module files")
	}
	for name, contents := range after {
		if strings.Contains(contents, "example.test/unused v0.0.0") {
			t.Fatalf("unused requirement remains in %s:\n%s", name, contents)
		}
	}
	output.Reset()
	diagnostics.Reset()
	if err := tidy(root, true, &output, &diagnostics); err != nil {
		t.Fatalf("check after tidy: %v\n%s\n%s", err, &output, &diagnostics)
	}
	if strings.Count(output.String(), "Checking module ") != 4 {
		t.Fatalf("check did not visit all modules: %s", &output)
	}
	if err := tidy(root, false, &output, &diagnostics); err != nil {
		t.Fatalf("second tidy: %v\n%s", err, &diagnostics)
	}
	if got := moduleFiles(t, root); !reflect.DeepEqual(got, after) {
		t.Fatal("tidy is not idempotent")
	}
	// A drift confined to any nested module must also fail the shared check.
	for _, directory := range []string{"_benchmarks", "_examples/simple"} {
		parent := ".."
		if strings.HasPrefix(directory, "_examples/") {
			parent = "../.."
		}
		writeFixture(t, root, directory+"/go.mod", nestedModule(filepath.Base(directory), parent))
		before = moduleFiles(t, root)
		output.Reset()
		if err := tidy(root, true, &output, &diagnostics); err == nil || !strings.Contains(err.Error(), directory) {
			t.Fatalf("check did not reject the untidy module %s: %v", directory, err)
		}
		if got := moduleFiles(t, root); !reflect.DeepEqual(got, before) {
			t.Fatal("nested-module check modified files")
		}
		if err := tidy(root, false, &output, &diagnostics); err != nil {
			t.Fatalf("repair nested module: %v\n%s", err, &diagnostics)
		}
	}
}

func TestRunFromNestedDirectory(t *testing.T) {
	root := fixture(t)
	t.Chdir(filepath.Join(root, "_examples", "simple"))
	var output, diagnostics bytes.Buffer
	if got := run([]string{"-check"}, &output, &diagnostics); got != 1 {
		t.Fatalf("untidy check exit %d: %s", got, &diagnostics)
	}
	if !strings.Contains(diagnostics.String(), "mise run tidy") {
		t.Fatalf("missing repair instructions: %s", &diagnostics)
	}
	if got := run(nil, &output, &diagnostics); got != 0 {
		t.Fatalf("tidy exit %d: %s", got, &diagnostics)
	}
	if got := run([]string{"-check"}, &output, &diagnostics); got != 0 {
		t.Fatalf("tidy check exit %d: %s", got, &diagnostics)
	}
}

func TestRunOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	var output, diagnostics bytes.Buffer
	if got := run(nil, &output, &diagnostics); got != 1 {
		t.Fatalf("outside repository exit %d: %s", got, &diagnostics)
	}
}

func TestRunArguments(t *testing.T) {
	for _, test := range []struct {
		args []string
		want int
	}{
		{[]string{"-help"}, 0},
		{[]string{"-unknown"}, 2},
		{[]string{"unexpected"}, 2},
	} {
		var output, diagnostics bytes.Buffer
		if got := run(test.args, &output, &diagnostics); got != test.want {
			t.Fatalf("run(%q) = %d; want %d: %s", test.args, got, test.want, &diagnostics)
		}
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "go.mod", "module github.com/echocat/cli4g\n\ngo 1.27.0\n\nrequire example.test/unused v0.0.0\n\nreplace example.test/unused => ./dependency\n")
	writeFixture(t, root, "root.go", "package root\n")
	writeFixture(t, root, "dependency/go.mod", "module example.test/unused\n\ngo 1.27.0\n")
	writeFixture(t, root, "dependency/dependency.go", "package dependency\n")
	for _, directory := range []string{"_benchmarks", "_examples/simple", "_examples/alpha"} {
		parent := ".."
		if strings.HasPrefix(directory, "_examples/") {
			parent = "../.."
		}
		writeFixture(t, root, directory+"/go.mod", nestedModule(filepath.Base(directory), parent))
		writeFixture(t, root, directory+"/main.go", "package main\n\nimport _ \"github.com/echocat/cli4g\"\n\nfunc main() {}\n")
	}
	return root
}

func nestedModule(name, parent string) string {
	return "module example.test/" + name + "\n\ngo 1.27.0\n\nrequire (\n github.com/echocat/cli4g v0.0.0\n example.test/unused v0.0.0 // indirect\n)\n\nreplace github.com/echocat/cli4g => " + parent + "\n\nreplace example.test/unused => " + parent + "/dependency\n"
}

func writeFixture(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func moduleFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "go.mod" || entry.Name() == "go.sum" {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			result[path] = string(contents)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}
