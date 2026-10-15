//go:build !unix

package codec

import (
	"io/fs"
	"os"
)

// Non-Unix platforms retain their native synchronous Open behavior. The caller
// validates the actual handle afterwards; there is no general cancellation claim.
func openRegularCodecFile(path string, flags int, permission fs.FileMode) (*os.File, error) {
	return os.OpenFile(path, flags, permission)
}
