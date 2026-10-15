package codec_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRequirementSerializationPreservesOpaquePathAndAllOptions(t *testing.T) {
	source := codec.NewFile()
	original := codec.FileRequirement{Path: "opaque-\xff-path", Mode: codec.FileReadWrite, Create: true, Append: true, Permission: 0640, AllowSpecial: true}
	plain, err := source.FormatRequirement(context.Background(), original)
	require.NoError(t, err)
	copy, err := source.ParseRequirement(context.Background(), codec.UsageDefault, plain...)
	require.NoError(t, err)
	assert.Equal(t, original, copy)
	_, err = source.Resolve(context.Background(), nil, original)
	assert.ErrorIs(t, err, codec.ErrInvalid)
}

func TestFileCodecSpecialPolicyIsExplicitInPathAndTypedRequirements(t *testing.T) {
	ctx := context.Background()
	source := codec.NewFile().WithAllowSpecial(true)
	requirement, err := source.ParseRequirement(ctx, codec.UsageFlag, os.DevNull)
	require.NoError(t, err)
	assert.True(t, requirement.AllowSpecial)
	scope := new(codec.ResourceScope)
	file, err := source.Resolve(ctx, scope, requirement)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.NoError(t, scope.Close())
	plain, err := source.FormatRequirement(ctx, requirement)
	require.NoError(t, err)
	copy, err := codec.NewFile().ParseRequirement(ctx, codec.UsageDefault, plain...)
	require.NoError(t, err)
	assert.Equal(t, requirement, copy)
	if runtime.GOOS != "windows" {
		scope := new(codec.ResourceScope)
		_, err = source.Resolve(ctx, scope, codec.FileRequirement{Path: os.DevNull, Mode: codec.FileRead})
		require.ErrorIs(t, err, codec.ErrSpecialFile, "typed requirements choose their own policy")
		require.NoError(t, scope.Close())
	}
}

func TestFileExplicitAppendTruncateAndSafeWrite(t *testing.T) {
	for _, test := range []struct {
		name          string
		append, trunc bool
		expected      string
	}{
		{"write", false, false, "new-existing"},
		{"append", true, false, "old-existingnew"},
		{"truncate", false, true, "new"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "output")
			require.NoError(t, os.WriteFile(path, []byte("old-existing"), 0600))
			scope := new(codec.ResourceScope)
			file, err := codec.NewFile().Resolve(context.Background(), scope, codec.FileRequirement{Path: path, Mode: codec.FileWrite, Append: test.append, Truncate: test.trunc})
			require.NoError(t, err)
			_, err = file.Write([]byte("new"))
			require.NoError(t, err)
			require.NoError(t, scope.Close())
			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(contents))
		})
	}
	path := filepath.Join(t.TempDir(), "created")
	scope := new(codec.ResourceScope)
	file, err := codec.NewFile(codec.FileReadWrite).Resolve(context.Background(), scope, codec.FileRequirement{Path: path, Mode: codec.FileReadWrite, Create: true})
	require.NoError(t, err)
	_, err = file.Write([]byte("data"))
	require.NoError(t, err)
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "data", string(contents))
	info, err := file.Stat()
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Zero(t, info.Mode().Perm()&0077, "default creation permissions are private")
	}
	require.NoError(t, scope.Close())
}

func TestRequirementChecksDynamicInterfacesAndCycles(t *testing.T) {
	file, err := os.Open(os.DevNull)
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()
	assert.ErrorIs(t, codec.ValidateRequirement(struct{ Value any }{file}), codec.ErrResourceRequirement)
	assert.ErrorIs(t, codec.ValidateRequirement(map[string]any{"file": file}), codec.ErrResourceRequirement)
	cycle := map[string]any{}
	cycle["self"] = cycle
	assert.NoError(t, codec.ValidateRequirement(cycle), "resource checking terminates; serializers must handle or reject cycles")
	cycle["file"] = file
	assert.ErrorIs(t, codec.ValidateRequirement(cycle), codec.ErrResourceRequirement)
}
