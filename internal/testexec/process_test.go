package testexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestProcessFixture(t *testing.T) {
	mode := os.Getenv("CLI4G_TESTEXEC_FIXTURE")
	if mode == "" {
		return
	}
	switch mode {
	case "exit":
		_, _ = fmt.Fprint(os.Stdout, "output\r\n")
		_, _ = fmt.Fprint(os.Stderr, "Error: failure\r\n")
		os.Exit(17)
	case "large":
		_, _ = fmt.Fprint(os.Stdout, strings.Repeat("out", 100_000))
		_, _ = fmt.Fprint(os.Stderr, strings.Repeat("err", 100_000))
		os.Exit(0)
	case "ready":
		for _, part := range []string{"rea", "dy", "\r", "\n"} {
			_, _ = fmt.Fprint(os.Stdout, part)
		}
		time.Sleep(30 * time.Second)
	case "ignore-term":
		ignoreTermination()
		_, _ = fmt.Fprintln(os.Stdout, "ready")
		time.Sleep(30 * time.Second)
	case "orphan-parent", "orphan-parent-error":
		child := exec.Command(os.Args[0], "-test.run=^TestProcessFixture$")
		child.Env = append(os.Environ(), "CLI4G_TESTEXEC_FIXTURE=orphan-child")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if mode == "orphan-parent-error" {
			os.Exit(17)
		}
		os.Exit(0)
	case "orphan-child":
		ignoreTermination()
		path := os.Getenv("CLI4G_TESTEXEC_HEARTBEAT")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			os.Exit(2)
		}
		defer func() { _ = file.Close() }()
		_, _ = fmt.Fprintln(os.Stdout, "child ready")
		deadline := time.After(30 * time.Second)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = file.WriteString("alive\n")
			case <-deadline:
				os.Exit(0)
			}
		}
	}
	os.Exit(0)
}

func fixture(t *testing.T, mode string) (string, []string, Options) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return executable, []string{"-test.run=^TestProcessFixture$"}, Options{Env: []string{"CLI4G_TESTEXEC_FIXTURE=" + mode}}
}

func TestRunPreservesExitAndBothChannels(t *testing.T) {
	executable, args, options := fixture(t, "exit")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result, err := Run(ctx, executable, args, options)
	if err != nil {
		t.Fatal(err)
	}
	if result != (Result{Code: 17, Stdout: "output\n", Stderr: "Error: failure\n"}) {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRunReadsChannelsConcurrently(t *testing.T) {
	executable, args, options := fixture(t, "large")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result, err := Run(ctx, executable, args, options)
	if err != nil || result.Code != 0 || result.Stdout != strings.Repeat("out", 100_000) || result.Stderr != strings.Repeat("err", 100_000) {
		t.Fatalf("result code=%d stdout=%d stderr=%d, error=%v", result.Code, len(result.Stdout), len(result.Stderr), err)
	}
}

func TestRunPreservesReadinessActionFailure(t *testing.T) {
	executable, args, options := fixture(t, "ready")
	cause := errors.New("readiness action failed")
	calls := 0
	options.ReadyStdout = regexp.MustCompile(`(?m)^ready\r?$`)
	options.OnReady = func(*os.Process) error { calls++; return cause }
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result, err := Run(ctx, executable, args, options)
	if !errors.Is(err, cause) || calls != 1 || result.Stdout != "ready\n" {
		t.Fatalf("calls=%d result=%#v error=%v", calls, result, err)
	}
}

func TestRunRejectsOutputOverflow(t *testing.T) {
	executable, args, options := fixture(t, "large")
	options.OutputLimit = 1024
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err := Run(ctx, executable, args, options)
	if err == nil || !strings.Contains(err.Error(), "output limit") {
		t.Fatalf("expected output limit failure, got %v", err)
	}
}

func TestRunStartFailure(t *testing.T) {
	_, err := Run(t.Context(), "cli4g-nonexistent-command-41938", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "start") {
		t.Fatalf("expected start failure, got %v", err)
	}
}

func TestRunRejectsMissingReadiness(t *testing.T) {
	executable, args, options := fixture(t, "exit")
	options.ReadyStdout = regexp.MustCompile(`not present`)
	_, err := Run(t.Context(), executable, args, options)
	if err == nil || !strings.Contains(err.Error(), "readiness") {
		t.Fatalf("expected readiness failure, got %v", err)
	}
}
