package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileResolvePreservesContentsUnlessTruncationIsExplicit(t *testing.T) {
	type options struct{ Output *os.File }
	for _, truncate := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "output")
		require.NoError(t, os.WriteFile(path, []byte("existing data"), 0600))
		var opened *os.File
		app := NewApplication[options]("app").
			WithErrorHandlerEnabled(false).
			WithFileFlag("output", func(f *FlagRequirement[*os.File, codec.FileRequirement]) {
				f.
					WithField("Output").
					WithDefault(codec.FileRequirement{Path: path, Mode: codec.FileWrite, Truncate: truncate})
			}).
			WithValidateHandler(func(context.Context, options) error { return NewValidationErrorf("invalid settings") }).
			WithCloseHandler(func(opts options) error { opened = opts.Output; return nil }).
			MustCreate()
		invocation, err := app.Parse(context.Background(), nil)
		require.ErrorIs(t, err, ErrValidation)
		assert.Nil(t, invocation)
		require.NotNil(t, opened)
		_, err = opened.Read(make([]byte, 1))
		assert.ErrorIs(t, err, os.ErrClosed)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		if truncate {
			assert.Empty(t, contents, "explicit truncation is a Resolve side effect, not a transaction")
		} else {
			assert.Equal(t, "existing data", string(contents), "failed validation must not implicitly truncate outputs")
		}
		require.NoError(t, app.Close())
	}
}
