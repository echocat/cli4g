//go:build unix

package testexec

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func prepareProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func terminateProcess(cmd *exec.Cmd, _ bool) error {
	term := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	if errors.Is(term, syscall.ESRCH) {
		term = nil
	}
	time.Sleep(250 * time.Millisecond)
	kill := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(kill, syscall.ESRCH) {
		kill = nil
	}
	return errors.Join(term, kill)
}

func CanInterrupt() bool                  { return true }
func Interrupt(process *os.Process) error { return process.Signal(os.Interrupt) }
