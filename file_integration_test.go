package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type trackedFileCodec struct {
	codec.Codec[*os.File, codec.FileRequirement]
	mu     *sync.Mutex
	files  *[]*os.File
	cancel context.CancelFunc
}

func (this trackedFileCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this trackedFileCodec) Resolve(ctx context.Context, scope codec.Scope, requirement codec.FileRequirement) (*os.File, error) {
	file, err := this.Codec.Resolve(ctx, scope, requirement)
	if file != nil {
		this.mu.Lock()
		*this.files = append(*this.files, file)
		this.mu.Unlock()
	}
	if this.cancel != nil {
		this.cancel()
	}
	return file, err
}

func TestFileCodecTypedAndStringDefaultsHelpAndLifecycle(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input")
	output := filepath.Join(dir, "output")
	require.NoError(t, os.WriteFile(input, []byte("payload"), 0600))
	require.NoError(t, os.WriteFile(output, []byte("prefix:"), 0600))
	type rootOptions struct{ Input *os.File }
	type childOptions struct{ Output *os.File }
	var files []*os.File
	var mu sync.Mutex
	tracking := func(mode codec.FileMode) trackedFileCodec {
		return trackedFileCodec{Codec: codec.NewFile(mode), mu: &mu, files: &files}
	}
	var helpOutput bytes.Buffer
	help, err := cli.NewTemplatedHelpHandler().WithOutput(&helpOutput).
		WithApplicationTemplate("{{range .Flags}}{{.Default}}{{end}}").Create()
	require.NoError(t, err)
	defaultInput := codec.FileRequirement{Path: input, Mode: codec.FileRead}
	app := cli.NewApplication[*rootOptions]().
		WithName("copy").
		WithErrorHandlerEnabled(false).
		WithHelpHandler(help).
		WithFileFlag("input", func(f *cli.FlagRequirement[*os.File, codec.FileRequirement]) {
			f.WithField("Input").
				WithCodec(tracking(codec.FileRead)).
				WithDefault(defaultInput)
		}).
		WithCommand(cli.NewCommand[*childOptions]().WithName("run").
			WithFileArgument("output", func(a *cli.ArgumentRequirement[*os.File, codec.FileRequirement]) {
				a.WithField("Output").
					WithCodec(tracking(codec.FileReadWrite)).
					WithDefaultStr(output)
			}).
			WithHandler(func(ctx context.Context, options *childOptions) error {
				invocation, _ := cli.InvocationFromContext(ctx)
				root := invocation.(*cli.Invocation[*rootOptions])
				_, optionsRoot := root.Application()
				if _, err := options.Output.Seek(0, io.SeekEnd); err != nil {
					return err
				}
				_, err := io.Copy(options.Output, optionsRoot.Input)
				return err
			}),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	defaultInput.Path = "changed after compilation"
	assert.Empty(t, files, "Create does not open defaults")
	definition, found := app.Flag("input")
	require.True(t, found)
	requirement, found, err := definition.Default(nil)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, input, requirement.(codec.FileRequirement).Path)
	formatted, _, err := definition.FormatDefault(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{input}, formatted)
	helpInvocation, err := app.Parse(nil, nil, "--help")
	require.NoError(t, err)
	require.NoError(t, helpInvocation.Execute(nil))
	require.NoError(t, helpInvocation.Close())
	assert.Contains(t, helpOutput.String(), input)
	assert.Empty(t, files, "Help and metadata never Resolve")
	invocation, err := app.Parse(nil, nil, "run")
	require.NoError(t, err)
	require.Len(t, files, 2)
	for _, file := range files {
		_, err := file.Stat()
		require.NoError(t, err)
	}
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	require.NoError(t, invocation.Close())
	for _, file := range files {
		_, err := file.Read(make([]byte, 1))
		assert.ErrorIs(t, err, os.ErrClosed)
	}
	contents, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "prefix:payload", string(contents), "ReadWrite does not implicitly truncate")
}

func TestFileCodecCleanupOnResolveValidationInitializationAndCancellation(t *testing.T) {
	for _, failure := range []string{"resolve", "validate", "initialize", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input")
			require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
			var files []*os.File
			var mu sync.Mutex
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			decoder := trackedFileCodec{Codec: codec.NewFile(), mu: &mu, files: &files}
			if failure == "cancel" {
				decoder.cancel = cancel
			}
			primary := errors.New("lifecycle failed")
			root := cli.NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
				WithFlag(cli.NewFlag(decoder).WithName("input").WithDefaultStr(path)) // unbound still owned
			if failure == "resolve" {
				root.WithFileFlag("second", func(f *cli.FlagRequirement[*os.File, codec.FileRequirement]) { f.WithDefaultStr(path + "-missing") })
			}
			if failure == "validate" {
				root.WithValidateHandler(func(context.Context, cli.NoOptions) error { return primary })
			}
			if failure == "initialize" {
				root.WithInitializeHandler(func(context.Context, cli.NoOptions) error { return primary })
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(ctx, nil)
			assert.Nil(t, invocation)
			require.Error(t, err)
			if failure == "cancel" {
				assert.ErrorIs(t, err, context.Canceled)
			} else if failure != "resolve" {
				assert.ErrorIs(t, err, primary)
			}
			require.Len(t, files, 1)
			_, err = files[0].Read(make([]byte, 1))
			assert.ErrorIs(t, err, os.ErrClosed)
		})
	}
}

func TestFileCodecChecksRequiredInputsBeforeOpeningAndSupportsParallelInvocations(t *testing.T) {
	dir := t.TempDir()
	created := filepath.Join(dir, "must-not-be-created")
	app, err := cli.NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
		WithFileFlag("output", func(f *cli.FlagRequirement[*os.File, codec.FileRequirement]) {
			f.WithCodec(codec.NewFile(codec.FileWrite)).WithDefaultStr(created)
		}).
		WithStringArgument("required", func(a *cli.ArgumentRequirement[string, string]) { a.WithRequired(true) }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	_, err = app.Parse(nil, nil)
	require.ErrorContains(t, err, "required argument")
	_, err = os.Stat(created)
	assert.ErrorIs(t, err, os.ErrNotExist)
	input := filepath.Join(dir, "input")
	require.NoError(t, os.WriteFile(input, []byte("content"), 0600))
	reader, err := cli.NewSimpleApplication().WithName("read").WithErrorHandlerEnabled(false).WithFileFlag("input").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, reader.Close()) }()
	results := make(chan error, 16)
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			invocation, err := reader.Parse(nil, nil, "--input", input)
			if err != nil {
				results <- err
				return
			}
			_, value := invocation.Flag("input")
			content, readErr := io.ReadAll(value.(*os.File))
			closeErr := invocation.Close()
			if string(content) != "content" {
				readErr = fmt.Errorf("unexpected content %q", content)
			}
			results <- errors.Join(readErr, closeErr)
		})
	}
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
}
