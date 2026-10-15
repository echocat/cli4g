//go:build !unix

package main

import "os/exec"

func nativeExitCode(exit *exec.ExitError) int { return exit.ExitCode() }
