package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRejectsAnonymousCodecResourceClosePromotion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
	for _, kind := range []string{"tag-flag", "tag-argument", "explicit-flag", "explicit-argument"} {
		t.Run(kind, func(t *testing.T) {
			var err error
			if kind == "tag-flag" {
				_, err = NewApplication[*struct {
					*os.File `flag:"input"`
				}]().WithName("app").WithCodec(codec.NewFile()).Create()
			} else if kind == "tag-argument" {
				_, err = NewApplication[*struct {
					*os.File `argument:"input"`
				}]().WithName("app").WithCodec(codec.NewFile()).Create()
			} else {
				root := NewApplication[*struct{ *os.File }]().WithName("app")
				if kind == "explicit-flag" {
					root.WithFileFlag("input", func(f *FlagRequirement[*os.File, codec.FileRequirement]) { f.WithField("File").WithDefaultStr(path) })
				} else {
					root.WithFileArgument("input", func(a *ArgumentRequirement[*os.File, codec.FileRequirement]) {
						a.WithField("File").WithDefaultStr(path)
					})
				}
				_, err = root.Create()
			}
			require.ErrorContains(t, err, "anonymous")
			require.ErrorContains(t, err, "Close")
		})
	}
}

func TestNamedFileBindingDoesNotPromoteScopeOwnedClose(t *testing.T) {
	type options struct {
		Input *os.File `flag:"input"`
	}
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
	app, err := NewApplication[*options]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).WithCodec(codec.NewFile()).Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--input", path)
	require.NoError(t, err)
	_, parsed := invocation.Application()
	_, err = parsed.Input.Stat()
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	require.NoError(t, invocation.Close())
	_, err = parsed.Input.Stat()
	assert.ErrorIs(t, err, os.ErrClosed)
}
