package main

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var shellEngine = flag.String("shell-engine", "", "run real container E2E tests with this engine")

// TestRunShellE2E measures the actual host-side orchestration as well as the
// instrumented fixture binaries. Ordinary package tests require no containers.
func TestRunShellE2E(t *testing.T) {
	if *shellEngine == "" {
		t.Skip("enable the real shell suite with -shell-engine")
	}
	require.NoError(t, run(t.Context(), []string{"--engine", *shellEngine, "--coverage"}, os.Stdout, os.Stderr))
}

func TestRunRejectsInvalidArgumentsBeforeStartingContainers(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"unexpected"}, {"--engine", "unsupported"}} {
		require.Error(t, run(t.Context(), args, io.Discard, io.Discard))
	}
}
