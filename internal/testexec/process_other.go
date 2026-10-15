//go:build !unix && !windows

package testexec

import (
	"errors"
	"os"
	"os/exec"
)

func prepareProcess(*exec.Cmd) {}
func terminateProcess(cmd *exec.Cmd, exited bool) error {
	if exited {
		return nil
	}
	err := cmd.Process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}
func CanInterrupt() bool          { return false }
func Interrupt(*os.Process) error { return errors.New("interrupt delivery is unsupported") }
