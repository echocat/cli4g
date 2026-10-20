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

//go:embed Containerfile Containerfile.powershell
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
	coverage := flags.Bool("coverage", false, "collect fixture coverage in profile.shells.cov")
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
	for _, name := range []string{"Containerfile", "Containerfile.powershell"} {
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
		build := []string{"build", "-mod=readonly", "-ldflags", ldflags}
		if *coverage {
			// The main package installs Go's runtime coverage hooks; the exported
			// report below selects only the actual library packages.
			build = append(build, "-cover", "-covermode=atomic", "-coverpkg=github.com/echocat/cli4g/internal/testdata/completion,github.com/echocat/cli4g,github.com/echocat/cli4g/codec")
		}
		build = append(build, "-o", filepath.Join(directory, "fixture-"+variant), "./internal/testdata/completion")
		if err := command(ctx, root, env, output, diagnostics, "go", build...); err != nil {
			return err
		}
	}
	env := append(exampleutil.NativeEnv(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
	if err := command(ctx, root, env, output, diagnostics, "go", "test", "-c", "-mod=readonly", "-tags=shell_e2e",
		"-o", filepath.Join(directory, "shell-e2e"), "./internal/shelltestcmd"); err != nil {
		return err
	}
	var coverageDirectories []string
	for _, file := range []string{"Containerfile", "Containerfile.powershell"} {
		coverageDirectory := ""
		if *coverage {
			coverageDirectory = filepath.Join(directory, "coverage-"+file)
			coverageDirectories = append(coverageDirectories, coverageDirectory)
		}
		if err := runContainer(ctx, selected, directory, file, coverageDirectory, output, diagnostics); err != nil {
			return err
		}
	}
	if *coverage {
		merged := filepath.Join(directory, "coverage-merged")
		if err := os.Mkdir(merged, 0700); err != nil {
			return err
		}
		env := exampleutil.NativeEnv()
		if err := command(ctx, root, env, output, diagnostics, "go", "tool", "covdata", "merge", "-pcombine", "-i="+strings.Join(coverageDirectories, ","), "-o="+merged); err != nil {
			return err
		}
		profile := filepath.Join(root, "profile.shells.cov")
		if err := command(ctx, root, env, output, diagnostics, "go", "tool", "covdata", "textfmt", "-pkg=github.com/echocat/cli4g,github.com/echocat/cli4g/codec", "-i="+merged, "-o="+profile); err != nil {
			return err
		}
		data, err := os.ReadFile(profile)
		if err != nil {
			return err
		}
		if len(strings.Split(strings.TrimSpace(string(data)), "\n")) < 2 {
			return fmt.Errorf("shell E2E produced no coverage data")
		}
		_, _ = fmt.Fprintln(output, "Shell fixture coverage:", profile)
	}
	return nil
}

func runContainer(ctx context.Context, selected, directory, file, coverageDirectory string, output, diagnostics io.Writer) error {
	idFile := filepath.Join(directory, file+".image-id")
	args := []string{"build", "--platform", "linux/" + runtime.GOARCH, "--file", filepath.Join(directory, file), "--iidfile", idFile}
	if file == "Containerfile.powershell" && runtime.GOARCH == "arm64" {
		args = append(args, "--build-arg", "POWERSHELL_IMAGE=mcr.microsoft.com/powershell:7.5-azurelinux-3.0-arm64@sha256:60d31793816646c4dd80b719b350ca68c8a53c000cbb2740655ddafe5e127903")
	}
	args = append(args, directory)
	if err := command(ctx, directory, nil, output, diagnostics, selected, args...); err != nil {
		return err
	}
	image, err := os.ReadFile(idFile)
	if err != nil {
		return err
	}
	name := filepath.Base(directory) + "-" + strings.ToLower(file)
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_ = command(cleanup, "", nil, io.Discard, io.Discard, selected, "rm", "--force", name)
	}()
	// Run by image ID rather than a mutable tag, so concurrent checkouts cannot
	// accidentally execute each other's test binaries. Runtime tests need no network.
	args = []string{"run", "--network=none", "--name", name}
	if coverageDirectory != "" {
		args = append(args, "--env", "GOCOVERDIR=/opt/coverage")
	} else {
		args = append(args, "--rm")
	}
	args = append(args, strings.TrimSpace(string(image)))
	if err := command(ctx, "", nil, output, diagnostics, selected, args...); err != nil {
		return err
	}
	if coverageDirectory != "" {
		if err := command(ctx, "", nil, output, diagnostics, selected, "cp", name+":/opt/coverage", coverageDirectory); err != nil {
			return err
		}
		entries, err := os.ReadDir(coverageDirectory)
		if err != nil {
			return err
		}
		var metadata, counters bool
		for _, entry := range entries {
			metadata = metadata || strings.HasPrefix(entry.Name(), "covmeta.")
			counters = counters || strings.HasPrefix(entry.Name(), "covcounters.")
		}
		if !metadata || !counters {
			return fmt.Errorf("shell fixtures produced no usable runtime coverage: %s", file)
		}
	}
	return nil
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
