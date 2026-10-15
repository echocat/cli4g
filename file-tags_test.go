package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileTagsOpenAndCloseBoundValues(t *testing.T) {
	type options struct {
		Input  *os.File `argument:"input" required:"true" fileMode:"read"`
		Output *os.File `flag:"output" required:"true" fileMode:"write"`
	}
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input"), filepath.Join(dir, "output")
	require.NoError(t, os.WriteFile(input, []byte("payload"), 0600))
	app := NewApplication[options]().WithName("files").WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := os.Stat(output)
	assert.ErrorIs(t, err, os.ErrNotExist, "compilation acquires no file resources")
	invocation, err := app.Parse(context.Background(), nil, "--output", output, input)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, opts := invocation.Application()
	data, err := io.ReadAll(opts.Input)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
	_, err = opts.Input.WriteString("not writable")
	require.Error(t, err)
	_, err = opts.Output.Write(data)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	for _, file := range []*os.File{opts.Input, opts.Output} {
		_, err = file.Stat()
		assert.ErrorIs(t, err, os.ErrClosed)
	}
	data, err = os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
}

func TestFileTagsSupportReadWriteAndSpecialOptIn(t *testing.T) {
	type options struct {
		File *os.File `flag:"file" required:"true" fileMode:"readWrite"`
		Null *os.File `flag:"null" allowSpecial:"true"`
	}
	path := filepath.Join(t.TempDir(), "data")
	require.NoError(t, os.WriteFile(path, []byte("abc"), 0600))
	app := NewApplication[options]().WithName("files").WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "--file", path, "--null", os.DevNull)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, opts := invocation.Application()
	data, err := io.ReadAll(opts.File)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(data))
	_, err = opts.File.Seek(0, io.SeekStart)
	require.NoError(t, err)
	_, err = opts.File.WriteString("X")
	require.NoError(t, err)
	data, err = io.ReadAll(opts.Null)
	require.NoError(t, err)
	assert.Empty(t, data)
	require.NoError(t, invocation.Close())
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Xbc", string(data), "readWrite does not implicitly truncate")
}

func TestTaggedFileSlicesAndImplicitReadCodec(t *testing.T) {
	type options struct {
		Files []*os.File `flag:"input" short:"i" fileMode:"read"`
		Other *os.File   `argument:"other" required:"true"`
	}
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
	require.NoError(t, os.WriteFile(first, []byte("one"), 0600))
	require.NoError(t, os.WriteFile(second, []byte("two"), 0600))
	app := NewApplication[options]().WithName("files").WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "-i", first, "-i", second, first)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, opts := invocation.Application()
	require.Len(t, opts.Files, 2)
	for index, file := range opts.Files {
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, []string{"one", "two"}[index], string(data))
	}
	require.NoError(t, invocation.Close())
	for _, file := range append(opts.Files, opts.Other) {
		_, err := file.Stat()
		assert.ErrorIs(t, err, os.ErrClosed)
	}
}

func TestFileTagsHelpDoesNotOpenOutput(t *testing.T) {
	type options struct {
		Output *os.File `flag:"output" fileMode:"write"`
	}
	path := filepath.Join(t.TempDir(), "not-created")
	var output bytes.Buffer
	help := NewTemplatedHelpHandler().WithOutput(&output).WithApplicationTemplate("{{.Usage}}").MustCreate()
	app := NewApplication[options]().WithName("files").WithHelpHandler(help).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "--output", path, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(context.Background()))
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, output.String(), "files")
}

type TaggedFileMixin struct {
	Input *os.File `argument:"input" required:"true" fileMode:"read"`
}

func TestFileTagsWorkInCommandPointerEmbeddings(t *testing.T) {
	type options struct{ *TaggedFileMixin }
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("command data"), 0600))
	app := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
		WithCommand(NewCommand[*options]().WithName("cat")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "cat", path)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	command, found := FindCommand[*options](app, "cat")
	require.True(t, found)
	opts, found := command.OptionsFrom(invocation)
	require.True(t, found)
	require.NotNil(t, opts.TaggedFileMixin)
	data, err := io.ReadAll(opts.Input)
	require.NoError(t, err)
	assert.Equal(t, "command data", string(data))
	require.NoError(t, invocation.Close())
	_, err = opts.Input.Stat()
	assert.ErrorIs(t, err, os.ErrClosed)
}

func TestInvalidFileTagsAreConfigurationErrors(t *testing.T) {
	for _, build := range []func() error{
		func() error {
			_, err := NewApplication[struct {
				File *os.File `flag:"file" fileMode:"invalid"`
			}]().WithName("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				File *os.File `flag:"file" fileMode:""`
			}]().WithName("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				File *os.File `flag:"file" allowSpecial:"invalid"`
			}]().WithName("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				Name string `flag:"name" fileMode:"read"`
			}]().WithName("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				Name string `argument:"name" allowSpecial:"true"`
			}]().WithName("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				File *os.File `flag:"file" fileMode:"read" type:"named"`
			}]().WithName("app").Create()
			return err
		},
	} {
		var err error
		require.NotPanics(t, func() { err = build() })
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrValidation)
	}
}

func TestExplicitFileBindingOverridesTagConfiguration(t *testing.T) {
	type options struct {
		File *os.File `flag:"old" fileMode:"invalid"`
	}
	app, err := NewApplication[options]().WithName("app").WithFileFlag("input", func(f *FlagRequirement[*os.File, codec.FileRequirement]) {
		f.WithField("File")
	}).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	_, found := app.Flag("old")
	assert.False(t, found)
	_, found = app.Flag("input")
	assert.True(t, found)
}

func TestRegistryFileCodecStillPrecedesImplicitFallback(t *testing.T) {
	type options struct {
		File *os.File `flag:"file"`
	}
	path := filepath.Join(t.TempDir(), "created")
	app := NewApplication[options]().WithName("app").WithErrorHandlerEnabled(false).
		WithCodec(codec.NewFile(codec.FileWrite)).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "--file", path)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, opts := invocation.Application()
	_, err = opts.File.WriteString("configured")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "configured", string(data))
}
