//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

func nativeExitCode(exit *exec.ExitError) int {
	if status, ok := exit.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return exit.ExitCode()
}
