// Command shelltestcmd runs completion E2E tests in an isolated Linux container.
// It prefers rootless Podman; --engine docker also supports Docker Desktop/rootless
// Docker. No repository mounts, daemon sockets, privileged containers or local
// shell installations are needed. Only static test binaries and embedded test
// assets are sent to the container builder.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/echocat/cli4g/internal/exampleutil"
)

//go:embed Containerfile
var assets embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, output, diagnostics io.Writer) error {
	flags := flag.NewFlagSet("shell-e2e", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	engine := flags.String("engine", "auto", "container engine: auto (Podman first), podman or docker")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected shell-e2e arguments")
	}
	selected, err := chooseEngine(*engine, exec.LookPath)
	if err != nil {
		return err
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("shell-e2e supports amd64 and arm64 hosts")
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Minute)
	defer cancel()
	if err := command(ctx, "", nil, io.Discard, diagnostics, selected, "info"); err != nil {
		return fmt.Errorf("container engine is unavailable: %w", err)
	}
	root, err := exampleutil.Root(".")
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "cli4g-shell-e2e-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(directory) }()
	for _, name := range []string{"Containerfile"} {
		data, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			return err
		}
	}
	for _, variant := range []string{"initial", "updated"} {
		public := "shell-setup"
		if variant == "updated" {
			public = "shell-admin"
		}
		ldflags := "-X main.variant=" + variant + " -X main.completionCommand=" + public + " -X main.completionRequestCommand=__suggest"
		env := append(exampleutil.NativeEnv(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
		if err := command(ctx, root, env, output, diagnostics, "go", "build", "-mod=readonly", "-ldflags", ldflags,
			"-o", filepath.Join(directory, "fixture-"+variant), "./internal/testdata/completion"); err != nil {
			return err
		}
	}
	env := append(exampleutil.NativeEnv(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
	if err := command(ctx, root, env, output, diagnostics, "go", "test", "-c", "-mod=readonly", "-tags=shell_e2e",
		"-o", filepath.Join(directory, "shell-e2e"), "./internal/shelltestcmd"); err != nil {
		return err
	}
	idFile := filepath.Join(directory, "image-id")
	if err := command(ctx, directory, nil, output, diagnostics, selected, "build", "--platform", "linux/"+runtime.GOARCH,
		"--file", filepath.Join(directory, "Containerfile"), "--iidfile", idFile, "--tag", "localhost/cli4g-shell-e2e:local", directory); err != nil {
		return err
	}
	image, err := os.ReadFile(idFile)
	if err != nil {
		return err
	}
	name := filepath.Base(directory)
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_ = command(cleanup, "", nil, io.Discard, io.Discard, selected, "rm", "--force", name)
	}()
	// Run by image ID rather than a mutable tag, so concurrent checkouts cannot
	// accidentally execute each other's test binaries. Runtime tests need no network.
	return command(ctx, "", nil, output, diagnostics, selected, "run", "--rm", "--network=none", "--name", name, strings.TrimSpace(string(image)))
}

func chooseEngine(value string, lookup func(string) (string, error)) (string, error) {
	if value != "auto" && value != "podman" && value != "docker" {
		return "", fmt.Errorf("unsupported container engine %q", value)
	}
	choices := []string{value}
	if value == "auto" {
		choices = []string{"podman", "docker"}
	}
	for _, choice := range choices {
		if path, err := lookup(choice); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("install Podman (recommended: rootless) or Docker to run shell E2E tests")
}

func command(ctx context.Context, directory string, env []string, output, diagnostics io.Writer, name string, args ...string) error {
	process := exec.CommandContext(ctx, name, args...)
	process.Dir = directory
	process.Env = append(process.Environ(), env...)
	process.Stdout, process.Stderr = output, diagnostics
	if err := process.Run(); err != nil {
		return fmt.Errorf("%s %q: %w", name, args, err)
	}
	return nil
}
