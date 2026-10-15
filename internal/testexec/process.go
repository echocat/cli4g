// Package testexec runs native test subprocesses with bounded output and cleanup.
package testexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Result struct {
	Code   int
	Stdout string
	Stderr string
}

type Options struct {
	Dir         string
	Env         []string
	ReadyStdout *regexp.Regexp
	OnReady     func(*os.Process) error
	OutputLimit int
}

type streamResult struct {
	text string
	err  error
}

// Run distinguishes program exit codes from infrastructure errors. Its context
// covers process exit AND EOF on both output pipes, including inherited pipes.
func Run(ctx context.Context, executable string, args []string, options Options) (Result, error) {
	result := Result{Code: -1}
	if ctx == nil {
		return result, errors.New("subprocess requires a context")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if options.OnReady != nil && options.ReadyStdout == nil {
		return result, errors.New("readiness action requires a stdout pattern")
	}
	limit := options.OutputLimit
	if limit <= 0 {
		limit = 8 * 1024 * 1024
	}
	cmd := exec.Command(executable, args...)
	cmd.Dir = options.Dir
	cmd.Env = append(cmd.Environ(), options.Env...)
	prepareProcess(cmd)
	stdout, stdoutWrite, err := os.Pipe()
	if err != nil {
		return result, err
	}
	defer func() { _ = stdout.Close() }()
	defer func() { _ = stdoutWrite.Close() }()
	stderr, stderrWrite, err := os.Pipe()
	if err != nil {
		return result, err
	}
	defer func() { _ = stderr.Close() }()
	defer func() { _ = stderrWrite.Close() }()
	cmd.Stdout, cmd.Stderr = stdoutWrite, stderrWrite
	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("start %s: %w", executable, err)
	}
	// Parent copies must not keep pipes open after the child and its descendants.
	_ = stdoutWrite.Close()
	_ = stderrWrite.Close()
	ready := make(chan struct{}, 1)
	outDone := make(chan streamResult, 1)
	errDone := make(chan streamResult, 1)
	waitDone := make(chan error, 1)
	go func() { outDone <- readStream(stdout, limit, options.ReadyStdout, ready) }()
	go func() { errDone <- readStream(stderr, limit, nil, nil) }()
	go func() { waitDone <- cmd.Wait() }()
	var waitErr, failure error
	var exited, readySeen bool
	outChannel, errChannel, waitChannel := outDone, errDone, waitDone
	for outChannel != nil || errChannel != nil || waitChannel != nil {
		select {
		case <-ctx.Done():
			failure = ctx.Err()
		case <-ready:
			readySeen = true
			ready = nil
			if options.OnReady != nil {
				failure = options.OnReady(cmd.Process)
			}
		case output := <-outChannel:
			result.Stdout = output.text
			outChannel = nil
			failure = output.err
		case output := <-errChannel:
			result.Stderr = output.text
			errChannel = nil
			failure = output.err
		case waitErr = <-waitChannel:
			exited = true
			waitChannel = nil
			result.Code = cmd.ProcessState.ExitCode()
		}
		if failure != nil {
			break
		}
	}
	if failure == nil && options.ReadyStdout != nil && !readySeen {
		// An already queued readiness event must not be lost when EOF wins select.
		select {
		case <-ready:
			if options.OnReady != nil {
				failure = options.OnReady(cmd.Process)
			}
		default:
			failure = errors.New("subprocess exited without its readiness message")
		}
	}
	if failure != nil {
		cleanupErr := terminateProcess(cmd, exited)
		// Closing our read ends unblocks readers even if a descendant escaped.
		_ = stdout.Close()
		_ = stderr.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for outChannel != nil || errChannel != nil || waitChannel != nil {
			select {
			case output := <-outChannel:
				result.Stdout = output.text
				outChannel = nil
			case output := <-errChannel:
				result.Stderr = output.text
				errChannel = nil
			case <-waitChannel:
				result.Code = cmd.ProcessState.ExitCode()
				waitChannel = nil
			case <-cleanupCtx.Done():
				return result, commandError(cmd, errors.Join(failure, cleanupErr, cleanupCtx.Err()))
			}
		}
		return result, commandError(cmd, errors.Join(failure, cleanupErr))
	}
	var exit *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exit) {
		return result, commandError(cmd, waitErr)
	}
	return result, nil
}

func readStream(file *os.File, limit int, pattern *regexp.Regexp, ready chan<- struct{}) streamResult {
	var output bytes.Buffer
	buffer := make([]byte, 32*1024)
	announced := false
	for {
		n, err := file.Read(buffer)
		if output.Len()+n > limit {
			return streamResult{normalize(output.String()), errors.New("subprocess output limit exceeded")}
		}
		_, _ = output.Write(buffer[:n])
		if !announced && pattern != nil && pattern.Match(output.Bytes()) {
			ready <- struct{}{}
			announced = true
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return streamResult{normalize(output.String()), err}
		}
	}
}

func normalize(output string) string { return strings.ReplaceAll(output, "\r\n", "\n") }

func commandError(cmd *exec.Cmd, err error) error {
	return fmt.Errorf("command %q in %q: %w", cmd.Args, cmd.Dir, err)
}
