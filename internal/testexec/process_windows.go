package testexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

func prepareProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminateProcess(cmd *exec.Cmd, exited bool) error {
	if exited {
		// taskkill cannot reliably identify descendants after the parent exits;
		// do not risk targeting a reused PID. Run still closes its own pipes.
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
	killErr := cmd.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	if err != nil {
		err = fmt.Errorf("process-tree termination: %w", err)
	}
	return errors.Join(err, killErr)
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var consoleProcesses = kernel32.NewProc("GetConsoleProcessList")
var consoleEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")

func CanInterrupt() bool {
	var process uint32
	count, _, _ := consoleProcesses.Call(uintptr(unsafe.Pointer(&process)), 1)
	return count != 0
}

// Ctrl+Break targets this process group. Go exposes it as os.Interrupt on Windows.
func Interrupt(process *os.Process) error {
	const ctrlBreakEvent = 1
	ok, _, err := consoleEvent.Call(ctrlBreakEvent, uintptr(process.Pid))
	if ok == 0 {
		return fmt.Errorf("send Ctrl+Break: %w", err)
	}
	return nil
}
