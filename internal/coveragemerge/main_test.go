package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRetainsHitsAcrossIndependentProfilesAndNormalizesExamples(t *testing.T) {
	profiles := []string{
		"mode: atomic\n" + modulePath + "/app.go:2.1,3.2 2 7\n" + modulePath + "/app.go:5.1,5.9 1 0\n",
		"mode: count\n" + modulePath + "/app.go:2.1,3.2 1 0\n./_examples/first/main.go:1.1,2.1 1 3\n",
		"mode: set\n" + modulePath + "/app.go:5.1,5.9 1 1\n" + modulePath + "/_examples/first/main.go:1.1,2.1 1 0\n",
	}
	want := "mode: set\n" + modulePath + "/_examples/first/main.go:1.1,2.1 1 1\n" + modulePath + "/app.go:2.1,3.2 2 1\n" + modulePath + "/app.go:5.1,5.9 1 1\n"
	for _, order := range [][]int{{0, 1, 2}, {2, 1, 0}} {
		blocks := make(map[position]block)
		for _, i := range order {
			require.NoError(t, readProfile(strings.NewReader(profiles[i]), blocks))
		}
		var result bytes.Buffer
		require.NoError(t, writeProfile(&result, blocks))
		assert.Equal(t, want, result.String())
	}
}

func TestReadRejectsInvalidProfiles(t *testing.T) {
	for _, input := range []string{
		"", "mode: unknown\napp.go:1.1,2.1 1 1\n", "mode: atomic\n", "mode: set\ninvalid\n",
		"mode: set\napp.go:0.1,2.1 1 1\n", "mode: set\napp.go:2.1,1.1 1 1\n",
		"mode: set\napp.go:1.3,1.2 1 1\n", "mode: set\n../outside.go:1.1,2.1 1 1\n",
		"mode: set\n/app.go:1.1,2.1 1 1\n", "mode: set\nC:\\app.go:1.1,2.1 1 1\n",
		"mode: set\napp.go:999999999999999999999.1,2.1 1 1\n", "mode: set\napp.go:1.1,2.1 1 18446744073709551616\n",
	} {
		t.Run(input, func(t *testing.T) {
			require.Error(t, readProfile(strings.NewReader(input), make(map[position]block)))
		})
	}
}

func TestCLIProducesOneValidProfileAndRejectsMissingInputs(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.cov")
	output := filepath.Join(directory, "merged.cov")
	require.NoError(t, os.WriteFile(input, []byte("mode: atomic\r\ncodec\\value.go:1.1,2.1 1 5\r\n"), 0600))
	var messages bytes.Buffer
	require.NoError(t, run([]string{"-output", output, input}, &messages, io.Discard))
	contents, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "mode: set\n"+modulePath+"/codec/value.go:1.1,2.1 1 1\n", string(contents))
	assert.Contains(t, messages.String(), "1 unique blocks")
	require.Error(t, run(nil, io.Discard, io.Discard))
	require.Error(t, run([]string{"-unknown"}, io.Discard, io.Discard))
	require.Error(t, run([]string{filepath.Join(directory, "missing.cov")}, io.Discard, io.Discard))
	require.Error(t, run([]string{"-output", directory, input}, io.Discard, io.Discard))
}
