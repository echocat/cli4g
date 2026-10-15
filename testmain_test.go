package cli

import (
	"os"
	"testing"
)

// Error-return tests opt out of process termination. Subprocess tests restore
// these captured production defaults when verifying the actual exit behavior.
var subprocessDefaultErrorHandler = DefaultErrorHandler

func TestMain(m *testing.M) {
	DefaultErrorHandler = nil
	os.Exit(m.Run())
}
