package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"golang.org/x/term"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type helpFDWriter struct {
	io.Writer
	fd uintptr
}

func (w helpFDWriter) Fd() uintptr { return w.fd }

func TestHelpOutputWidth(t *testing.T) {
	var size, calls int
	var failure error
	probe := func(fd int) (int, int, error) {
		calls++
		assert.Equal(t, 42, fd)
		return size, 25, failure
	}
	output := helpFDWriter{Writer: io.Discard, fd: 42}
	assert.Equal(t, 100, helpOutputWidth(output, 100, probe))
	assert.Zero(t, calls, "fixed width must not query the terminal")
	for _, writer := range []io.Writer{nil, io.Discard, new(bytes.Buffer)} {
		assert.Equal(t, 80, helpOutputWidth(writer, 0, probe))
	}
	assert.Zero(t, calls, "writers without Fd must not query another process stream")
	size = 120
	assert.Equal(t, 120, helpOutputWidth(output, 0, probe))
	size = 40
	assert.Equal(t, 40, helpOutputWidth(output, 0, probe), "probe for every rendering, not at Create")
	for _, size = range []int{0, -1} {
		assert.Equal(t, 80, helpOutputWidth(output, 0, probe))
	}
	size, failure = 120, errors.New("terminal size unavailable")
	assert.Equal(t, 80, helpOutputWidth(output, 0, probe))
}

func TestHelpOutputWidthFallsBackForFilesAndClosedDescriptors(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "help")
	require.NoError(t, err)
	assert.Equal(t, 80, helpOutputWidth(output, 0, term.GetSize))
	require.NoError(t, output.Close())
	assert.Equal(t, 80, helpOutputWidth(output, 0, term.GetSize))
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, reader.Close()) })
	t.Cleanup(func() { assert.NoError(t, writer.Close()) })
	assert.Equal(t, 80, helpOutputWidth(writer, 0, term.GetSize))
}

type helpNoProbeWriter struct{ bytes.Buffer }

func (*helpNoProbeWriter) Fd() uintptr { panic("terminal must not be queried") }

func TestHelpFormattingDisabledAndFixedWidthSkipTerminalProbe(t *testing.T) {
	app := NewSimpleApplication("app").MustCreate()
	t.Cleanup(func() { assert.NoError(t, app.Close()) })
	invocation := app.MustParse(t.Context(), nil, "--help")
	t.Cleanup(func() { assert.NoError(t, invocation.Close()) })
	for _, formatting := range []bool{false, true} {
		var output helpNoProbeWriter
		r := NewTemplatedHelpHandler().WithOutput(&output).
			WithApplicationTemplate("literal\ttext\n").WithFormatting(formatting)
		if formatting {
			r.WithWidth(80)
		}
		require.NoError(t, r.MustCreate()(t.Context(), invocation))
		if formatting {
			assert.Equal(t, "literal  text\n", output.String())
		} else {
			assert.Equal(t, "literal\ttext\n", output.String())
		}
	}
}
