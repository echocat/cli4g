package exampleutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootModuleLineEndings(t *testing.T) {
	for _, test := range []struct {
		name   string
		module string
	}{
		{"LF", "module github.com/echocat/cli4g\n\ngo 1.27.0\n"},
		{"CRLF", "module github.com/echocat/cli4g\r\n\r\ngo 1.27.0\r\n"},
		{"no final newline", "module github.com/echocat/cli4g"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			requireWriteModule(t, root, test.module)
			child := filepath.Join(root, "_examples", "simple")
			requireWriteModule(t, child, "module github.com/echocat/cli4g/_examples/simple\n")
			got, err := Root(child)
			if err != nil {
				t.Fatal(err)
			}
			if got != root {
				t.Fatalf("root: got %q, want %q", got, root)
			}
		})
	}
}

func TestRootRejectsDifferentModule(t *testing.T) {
	root := t.TempDir()
	requireWriteModule(t, root, "module github.com/echocat/cli4g-other\r\n")
	if _, err := Root(root); err == nil {
		t.Fatal("a module sharing only the prefix must not be accepted")
	}
}

func requireWriteModule(t *testing.T, directory, contents string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
