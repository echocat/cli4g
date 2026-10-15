//go:build linux

package codec_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fifoCleanupScope struct {
	codec.ResourceScope
	registered, closed int
}

func (scope *fifoCleanupScope) OnClose(release func() error) {
	scope.registered++
	scope.ResourceScope.OnClose(func() error { scope.closed++; return release() })
}

// Always provide a peer when ending a blocking-open test, even on failure. This
// prevents a regression in opening strategy from leaving a test goroutine stuck.
func releaseFIFOOpen(t *testing.T, path string, result <-chan error) {
	t.Helper()
	peer, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
	require.NoError(t, err)
	defer func() { require.NoError(t, peer.Close()) }()
	select {
	case <-result:
	case <-time.After(2 * time.Second):
		t.Fatal("FIFO open did not return after peer was provided")
	}
}
func TestDefaultFileCodecRejectsFIFOWithoutWaitingForWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	require.NoError(t, syscall.Mkfifo(path, 0600))
	scope := new(fifoCleanupScope)
	result := make(chan error, 1)
	go func() {
		_, err := codec.NewFile().Resolve(context.Background(), scope, codec.FileRequirement{Path: path, Mode: codec.FileRead})
		result <- err
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, codec.ErrSpecialFile)
	case <-time.After(time.Second):
		releaseFIFOOpen(t, path, result)
		t.Fatal("default FIFO open waited for a writer")
	}
	require.NoError(t, scope.Close())
	assert.Equal(t, 1, scope.registered)
	assert.Equal(t, 1, scope.closed, "rejected FIFO handle is still scope-owned")
}
func TestExplicitFIFOOptInDocumentsSynchronousCancellationLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	require.NoError(t, syscall.Mkfifo(path, 0600))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scope := new(fifoCleanupScope)
	result := make(chan error, 1)
	go func() {
		_, err := codec.NewFile().WithAllowSpecial(true).Resolve(ctx, scope, codec.FileRequirement{Path: path, Mode: codec.FileRead, AllowSpecial: true})
		result <- err
	}()
	select {
	case err := <-result:
		t.Fatalf("expected a synchronous FIFO wait before cancellation, got %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-result:
		t.Fatalf("an already blocking native FIFO open unexpectedly returned without a peer: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	peer, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
	require.NoError(t, err)
	defer func() { require.NoError(t, peer.Close()) }()
	select {
	case err := <-result:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("FIFO open did not return after peer was provided")
	}
	require.NoError(t, scope.Close())
	require.NoError(t, scope.Close())
	assert.Equal(t, 1, scope.registered)
	assert.Equal(t, 1, scope.closed, "canceled open cleans the acquired handle once after returning")
}
