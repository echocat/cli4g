// Package exampleutil shares example discovery between Go tests and the runner.
package exampleutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

func Root(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		module, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err == nil {
			line, _, _ := strings.Cut(string(module), "\n")
			if strings.TrimSpace(line) == "module github.com/echocat/cli4g" {
				return directory, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("cannot find cli4g repository from %q", start)
		}
		directory = parent
	}
}

func Names(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "_examples"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := Directory(root, entry.Name()); err != nil {
			return nil, err
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no example modules found")
	}
	return names, nil // os.ReadDir sorts by name.
}

func Directory(root, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("invalid example name %q", name)
	}
	directory := filepath.Join(root, "_examples", name)
	if _, err := os.Stat(filepath.Join(directory, "go.mod")); err != nil {
		return "", fmt.Errorf("example %q has no module: %w", name, err)
	}
	return directory, nil
}

// NativeEnv isolates module/workspace and target settings without changing the
// caller's persistent Go configuration, PATH, proxy or cache settings.
func NativeEnv() []string {
	return []string{
		"GOOS=" + runtime.GOOS, "GOARCH=" + runtime.GOARCH, "GOWORK=off",
		"GOAMD64=v1", "GOARM64=v8.0", "GO386=sse2", "GOARM=7",
		"GOPPC64=power8", "GOMIPS=hardfloat", "GOMIPS64=hardfloat",
		"GORISCV64=rva20u64",
	}
}
