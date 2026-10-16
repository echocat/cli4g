package cli

import "io"

const defaultHelpWidth = 80

// Probe only the selected writer, never another process stream. Buffers,
// redirections, closed descriptors and unsupported terminals all use the fallback.
// The probe parameter keeps failure and platform-specific behavior testable
// without mutable globals or an interactive terminal.
func helpOutputWidth(output io.Writer, configured int, getSize func(int) (int, int, error)) int {
	if configured > 0 {
		return configured
	}
	if file, ok := output.(interface{ Fd() uintptr }); ok {
		if width, _, err := getSize(int(file.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return defaultHelpWidth
}
