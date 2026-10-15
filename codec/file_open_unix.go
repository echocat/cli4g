//go:build unix

package codec

import (
	"io/fs"
	"os"
	"syscall"
)

// Nonblocking open avoids waiting for a FIFO peer before the opened handle's
// type is checked. Regular files ignore O_NONBLOCK. This is not a promise that
// every kernel/device/filesystem Open or Stat is context-interruptible.
func openRegularCodecFile(path string, flags int, permission fs.FileMode) (*os.File, error) {
	// #nosec G304 -- This CLI codec intentionally opens caller-selected paths; it is not a filesystem sandbox.
	return os.OpenFile(path, flags|syscall.O_NONBLOCK, permission)
}
