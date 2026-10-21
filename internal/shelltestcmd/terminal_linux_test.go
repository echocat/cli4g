//go:build linux && shell_e2e

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// terminal connects a real interactive shell to Linux's PTY subsystem. The host
// cross-builds these tests for the container; normal go test runs exclude them.
type terminal struct {
	t          *testing.T
	shell      string
	process    *exec.Cmd
	master     *os.File
	pending    []byte
	transcript []byte
}

func startTerminal(t *testing.T, shell string, args, env []string, directory string) *terminal {
	t.Helper()
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	must(t, err)
	master := os.NewFile(uintptr(fd), "pty-master")
	t.Cleanup(func() { _ = master.Close() })
	must(t, unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	must(t, err)
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	must(t, err)
	defer func() { _ = slave.Close() }()
	must(t, unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 160}))
	process := exec.Command(shell, args...)
	process.Env, process.Dir = env, directory
	process.Stdin, process.Stdout, process.Stderr = slave, slave, slave
	process.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	must(t, process.Start())
	terminal := &terminal{t: t, shell: shell, process: process, master: master}
	t.Cleanup(terminal.close)
	return terminal
}

func (terminal *terminal) send(value string) {
	terminal.t.Helper()
	_, err := terminal.master.WriteString(value)
	must(terminal.t, err)
}

func (terminal *terminal) expect(pattern string) []string {
	terminal.t.Helper()
	matcher := regexp.MustCompile("(?s)" + pattern)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if indexes := matcher.FindSubmatchIndex(terminal.pending); indexes != nil {
			result := make([]string, len(indexes)/2)
			for index := range result {
				start, end := indexes[index*2], indexes[index*2+1]
				if start >= 0 {
					result[index] = string(terminal.pending[start:end])
				}
			}
			terminal.pending = terminal.pending[indexes[1]:]
			return result
		}
		must(terminal.t, terminal.master.SetReadDeadline(deadline))
		var data [65536]byte
		count, err := terminal.master.Read(data[:])
		if count > 0 {
			terminal.transcript = append(terminal.transcript, data[:count]...)
			terminal.pending = append(terminal.pending, data[:count]...)
			// Reedline waits for a cursor-position reply before drawing its prompt.
			queries := bytes.Count(terminal.pending, []byte("\x1b[6n"))
			if queries > 0 {
				terminal.pending = bytes.ReplaceAll(terminal.pending, []byte("\x1b[6n"), nil)
				terminal.send(string(bytes.Repeat([]byte("\x1b[1;1R"), queries)))
			}
		}
		if err != nil {
			terminal.t.Fatalf("terminal did not produce %q: %v\ntranscript: %q", pattern, err, terminal.transcript)
		}
	}
}

func (terminal *terminal) prompt() {
	terminal.t.Helper()
	terminal.expect("E2E> ")
	if terminal.shell == "nu" {
		terminal.expect(regexp.QuoteMeta("\x1b[?25h"))
	}
}

func (terminal *terminal) complete(line string) string { return terminal.buffer(line, "\t", -1) }

func (terminal *terminal) buffer(line, keys string, cursor int) string {
	terminal.t.Helper()
	terminal.send("\x01\x0b" + line)
	if cursor >= 0 {
		terminal.send("\x01" + string(bytes.Repeat([]byte{0x06}, cursor)))
	}
	terminal.send(keys + "\x0f")
	matched := terminal.expect("\r?\n__CLI4G_BUFFER__(.*?)__CLI4G_END__")
	var value string
	must(terminal.t, json.Unmarshal([]byte(matched[1]), &value))
	terminal.prompt()
	return value
}

func (terminal *terminal) close() {
	_ = unix.Kill(-terminal.process.Process.Pid, unix.SIGTERM)
	_ = terminal.master.Close()
	done := make(chan error, 1)
	go func() { done <- terminal.process.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = unix.Kill(-terminal.process.Process.Pid, unix.SIGKILL)
		<-done
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
