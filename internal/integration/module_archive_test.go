package integration

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"

	"github.com/echocat/cli4g/internal/exampleutil"
)

var checkModuleArchive = flag.Bool("module-archive", false, "test the distributable Go module without nested modules or Git metadata")

func TestModuleArchive(t *testing.T) {
	if !*checkModuleArchive {
		t.Skip("enable module archive checks with -module-archive")
	}
	root, err := exampleutil.Root(".")
	if err != nil {
		t.Fatal(err)
	}
	// Include intended repository sources and new fixes, but not ignored local
	// artifacts. The Go module zip writer then applies the distribution rules.
	listed := runTool(t, root, nil, "git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	var files []modzip.File
	for _, name := range strings.Split(strings.TrimSuffix(listed.Stdout, "\x00"), "\x00") {
		path := filepath.Join(root, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue // A working-tree deletion or rename may still be in the index.
		}
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, moduleSourceFile{name: name, path: path, info: info})
	}
	version := module.Version{Path: "github.com/echocat/cli4g", Version: "v0.0.0"}
	archivePath := filepath.Join(t.TempDir(), "module.zip")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	createErr := modzip.Create(archive, version, files)
	closeErr := archive.Close()
	if createErr != nil {
		t.Fatal(createErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	extracted := filepath.Join(t.TempDir(), "module")
	if err := modzip.Unzip(extracted, version, archivePath); err != nil {
		t.Fatal(err)
	}
	for _, omitted := range []string{".git", "_examples/simple/go.mod", "_benchmarks/go.mod"} {
		if _, err := os.Stat(filepath.Join(extracted, filepath.FromSlash(omitted))); !os.IsNotExist(err) {
			t.Fatalf("module archive must omit %s: %v", omitted, err)
		}
	}
	// This builds and tests the whole module. A cold CI cache and concurrent
	// quality checks can exceed the short per-tool budget used by examples.
	runToolContextWithTimeout(t, t.Context(), 10*time.Minute, extracted, exampleutil.NativeEnv(), "go", "test", "-count=1", "./...")
}

type moduleSourceFile struct {
	name string
	path string
	info os.FileInfo
}

func (file moduleSourceFile) Path() string { return file.name }

func (file moduleSourceFile) Lstat() (os.FileInfo, error) { return file.info, nil }

func (file moduleSourceFile) Open() (io.ReadCloser, error) { return os.Open(file.path) }
