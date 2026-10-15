package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestListFormats(t *testing.T) {
	root := t.TempDir()
	names := []string{"alpha", "simple"}
	for _, name := range names {
		directory := filepath.Join(root, "_examples", name)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example/"+name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var output, diagnostics bytes.Buffer
	if code := list(root, []string{"-F", "json"}, &output, &diagnostics); code != 0 {
		t.Fatalf("exit %d: %s", code, diagnostics.String())
	}
	var got []string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil || !slices.Equal(got, names) {
		t.Fatalf("unexpected list %q: %v", output.String(), err)
	}
	output.Reset()
	if code := list(root, []string{"--format", "invalid"}, &output, &diagnostics); code != 2 {
		t.Fatalf("invalid format exit %d", code)
	}
}

func TestRunRejectsInvalidNames(t *testing.T) {
	var output, diagnostics bytes.Buffer
	for _, name := range []string{"../simple", "not-an-example"} {
		if code := run([]string{"run", name}, nil, &output, &diagnostics); code != 2 {
			t.Fatalf("name %q: exit %d", name, code)
		}
	}
}
