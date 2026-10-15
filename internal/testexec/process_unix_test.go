//go:build unix

package testexec

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"
)

func ignoreTermination() { signal.Ignore(syscall.SIGTERM) }

func TestRunDeadlineTerminatesResistantProcess(t *testing.T) {
	executable, args, options := fixture(t, "ignore-term")
	options.ReadyStdout = regexp.MustCompile(`(?m)^ready$`)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	started := time.Now()
	result, err := Run(ctx, executable, args, options)
	if !errors.Is(err, context.DeadlineExceeded) || result.Stdout != "ready\n" || time.Since(started) > 7*time.Second {
		t.Fatalf("deadline result=%#v error=%v", result, err)
	}
}

func TestRunDeadlineTerminatesDescendantsAfterParentExit(t *testing.T) {
	for _, mode := range []string{"orphan-parent", "orphan-parent-error"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "heartbeat")
			executable, args, options := fixture(t, mode)
			options.Env = append(options.Env, "CLI4G_TESTEXEC_HEARTBEAT="+path)
			options.ReadyStdout = regexp.MustCompile(`(?m)^child ready$`)
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			result, err := Run(ctx, executable, args, options)
			if !errors.Is(err, context.DeadlineExceeded) || result.Stdout != "child ready\n" {
				t.Fatalf("descendant result=%#v error=%v", result, err)
			}
			before, err := os.Stat(path)
			if err != nil || before.Size() == 0 {
				t.Fatalf("heartbeat missing: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			after, err := os.Stat(path)
			if err != nil || after.Size() != before.Size() {
				t.Fatalf("descendant is still running: %v", err)
			}
		})
	}
}
