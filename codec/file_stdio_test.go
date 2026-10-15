package codec_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stdioCleanupScope struct {
	codec.ResourceScope
	registrations int
}

func (scope *stdioCleanupScope) OnClose(release func() error) {
	scope.registrations++
	scope.ResourceScope.OnClose(release)
}

func TestFileStdioAliasesAreBorrowedAndPreserved(t *testing.T) {
	ctx := codec.ContextWithStdio(context.Background(), os.Stdin, os.Stdout, os.Stderr)
	for _, test := range []struct {
		name string
		mode codec.FileMode
		path string
		file *os.File
	}{
		{"read-dash", codec.FileRead, "-", os.Stdin},
		{"read-stdin", codec.FileRead, "stdin", os.Stdin},
		{"write-dash", codec.FileWrite, "-", os.Stdout},
		{"write-stdout", codec.FileWrite, "stdout", os.Stdout},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := codec.NewFile(test.mode).WithAllowStdio(true)
			requirement, err := source.ParseRequirement(ctx, codec.UsageFlag, test.path)
			require.NoError(t, err)
			assert.True(t, requirement.AllowStdio)
			assert.False(t, requirement.AllowSpecial, "stdio does not require nonregular-file access")
			plain, err := source.FormatRequirement(ctx, requirement)
			require.NoError(t, err)
			copy, err := codec.NewFile().ParseRequirement(ctx, codec.UsageDefault, plain...)
			require.NoError(t, err)
			assert.Equal(t, requirement, copy)
			clone, err := source.CloneRequirement(ctx, requirement)
			require.NoError(t, err)
			assert.Equal(t, requirement, clone)
			display, err := source.DisplayRequirement(ctx, requirement)
			require.NoError(t, err)
			assert.Equal(t, []string{test.path}, display)
			scope := new(stdioCleanupScope)
			file, err := codec.NewFile().Resolve(ctx, scope, copy)
			require.NoError(t, err)
			assert.Same(t, test.file, file)
			require.Zero(t, scope.registrations, "borrowed streams must not be registered for cleanup")
			require.NoError(t, scope.Close())
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			_, err = source.Resolve(canceled, scope, requirement)
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestFileStdioRequiresContextFiles(t *testing.T) {
	var nilFile *os.File
	for _, ctx := range []context.Context{
		context.Background(),
		codec.ContextWithStdio(nil, nil, nil, nil),
		codec.ContextWithStdio(nil, nilFile, nilFile, nil),
		codec.ContextWithStdio(nil, bytes.NewBufferString("input"), new(bytes.Buffer), nil),
	} {
		for _, mode := range []codec.FileMode{codec.FileRead, codec.FileWrite} {
			scope := new(stdioCleanupScope)
			file, err := codec.NewFile(mode).Resolve(ctx, scope, codec.FileRequirement{Path: "-", Mode: mode, AllowStdio: true})
			assert.Nil(t, file)
			assert.ErrorIs(t, err, codec.ErrStdioUnavailable)
			assert.Zero(t, scope.registrations)
		}
	}
}

func TestFileStdioRejectsIncompatibleRequirements(t *testing.T) {
	ctx := context.Background()
	for _, requirement := range []codec.FileRequirement{
		{Path: "stdout", Mode: codec.FileRead},
		{Path: "stdin", Mode: codec.FileWrite},
		{Path: "-", Mode: codec.FileReadWrite},
		{Path: "stdin", Mode: codec.FileReadWrite},
		{Path: "stdout", Mode: codec.FileReadWrite},
		{Path: "stdout", Mode: codec.FileWrite, Append: true},
		{Path: "-", Mode: codec.FileWrite, Truncate: true},
	} {
		requirement.AllowStdio = true
		source := codec.NewFile(requirement.Mode).WithAllowStdio(true)
		if !requirement.Append && !requirement.Truncate {
			_, err := source.ParseRequirement(ctx, codec.UsageArgument, requirement.Path)
			assert.ErrorIs(t, err, codec.ErrIllegalArgument)
		}
		scope := new(stdioCleanupScope)
		_, err := source.Resolve(ctx, scope, requirement)
		assert.ErrorIs(t, err, codec.ErrIllegalArgument)
		assert.Zero(t, scope.registrations)
		_, err = source.FormatRequirement(ctx, requirement)
		assert.ErrorIs(t, err, codec.ErrIllegalArgument)
	}
}

func TestFileStdioLiteralPathsAndTypedPolicy(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	for _, path := range []string{"-", "stdin", "stdout", "STDIN", "Stdout"} {
		require.NoError(t, os.WriteFile(path, []byte("literal"), 0600))
		for _, allowStdio := range []bool{false, true} {
			input := path
			if allowStdio {
				input = "./" + path
			}
			// Even AllowSpecial does not implicitly enable stdio aliases.
			source := codec.NewFile().WithAllowSpecial(true).WithAllowStdio(allowStdio)
			requirement, err := source.ParseRequirement(ctx, codec.UsageFlag, input)
			require.NoError(t, err)
			scope := new(codec.ResourceScope)
			file, err := source.Resolve(ctx, scope, requirement)
			require.NoError(t, err)
			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, "literal", string(content))
			require.NoError(t, scope.Close())
			_, err = file.Stat()
			assert.ErrorIs(t, err, os.ErrClosed)
		}
	}
	// Typed requirements use their own policy, not the codec's defaults.
	source := codec.NewFile().WithAllowStdio(true)
	scope := new(codec.ResourceScope)
	file, err := source.Resolve(ctx, scope, codec.FileRequirement{Path: "stdin"})
	require.NoError(t, err)
	assert.NotSame(t, os.Stdin, file)
	require.NoError(t, scope.Close())
	// Enabling the policy keeps ordinary (case-sensitive) paths unchanged.
	for _, path := range []string{"STDIN", "Stdout"} {
		scope := new(codec.ResourceScope)
		file, err := source.Resolve(ctx, scope, codec.FileRequirement{Path: path, AllowStdio: true})
		require.NoError(t, err)
		assert.NotSame(t, os.Stdin, file)
		require.NoError(t, scope.Close())
	}
}
