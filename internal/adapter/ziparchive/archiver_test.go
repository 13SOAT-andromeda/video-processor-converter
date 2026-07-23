package ziparchive_test

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ziparchive"
)

func TestZip(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "frame_0001.jpg"), []byte("aaa"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "frame_0002.jpg"), []byte("bbb"), 0o644))

	destZip := filepath.Join(t.TempDir(), "out.zip")
	err := ziparchive.NewArchiver().Zip(context.Background(), srcDir, destZip)
	require.NoError(t, err)

	r, err := zip.OpenReader(destZip)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	assert.ElementsMatch(t, []string{"frame_0001.jpg", "frame_0002.jpg"}, names)
}

func TestZipEmptyDirProducesEmptyZip(t *testing.T) {
	destZip := filepath.Join(t.TempDir(), "out.zip")
	err := ziparchive.NewArchiver().Zip(context.Background(), t.TempDir(), destZip)
	require.NoError(t, err)

	r, err := zip.OpenReader(destZip)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	assert.Empty(t, r.File)
}

func TestZipBadGlobPatternReturnsError(t *testing.T) {
	destZip := filepath.Join(t.TempDir(), "out.zip")
	err := ziparchive.NewArchiver().Zip(context.Background(), "[", destZip)
	assert.Error(t, err)
}

func TestZipDestPathIsDirectoryReturnsError(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("aaa"), 0o644))

	// destZipPath aponta pra um diretório existente: os.Create falha.
	err := ziparchive.NewArchiver().Zip(context.Background(), srcDir, t.TempDir())
	assert.Error(t, err)
}

func TestZipSkipsSubdirectories(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "frame_0001.jpg"), []byte("aaa"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(srcDir, "subdir"), 0o755))

	destZip := filepath.Join(t.TempDir(), "out.zip")
	err := ziparchive.NewArchiver().Zip(context.Background(), srcDir, destZip)
	require.NoError(t, err)

	r, err := zip.OpenReader(destZip)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	assert.Equal(t, []string{"frame_0001.jpg"}, names)
}

func TestZipDanglingSymlinkReturnsError(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.Symlink(filepath.Join(srcDir, "does-not-exist"), filepath.Join(srcDir, "broken")))

	destZip := filepath.Join(t.TempDir(), "out.zip")
	err := ziparchive.NewArchiver().Zip(context.Background(), srcDir, destZip)
	assert.Error(t, err)
}

func TestZipCancelledContext(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("aaa"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ziparchive.NewArchiver().Zip(ctx, srcDir, filepath.Join(t.TempDir(), "out.zip"))
	assert.ErrorIs(t, err, context.Canceled)
}
