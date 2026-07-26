package ffmpeg_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffmpeg"
)

// Usa o ffmpeg real (instalado no runner de CI e localmente) contra os
// fixtures em test/fixtures — não é teste de integração: não toca AWS/LocalStack.

func TestExtractFramesReturnsFrameCount(t *testing.T) {
	outDir := t.TempDir()
	n, err := ffmpeg.NewExtractor(1).ExtractFrames(context.Background(), "../../../test/fixtures/sample_1080p.mp4", outDir)
	require.NoError(t, err)
	assert.Positive(t, n)

	frames, err := filepath.Glob(filepath.Join(outDir, "*.jpg"))
	require.NoError(t, err)
	assert.Len(t, frames, n)
}

func TestExtractFramesInvalidVideoReturnsError(t *testing.T) {
	_, err := ffmpeg.NewExtractor(1).ExtractFrames(context.Background(), "../../../test/fixtures/invalid_video.mp4", t.TempDir())
	assert.Error(t, err)
}

func TestExtractFramesMkdirFailure(t *testing.T) {
	// outDir aponta pra dentro de um arquivo (não diretório): MkdirAll falha com ENOTDIR.
	notADir := filepath.Join(t.TempDir(), "im-a-file")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o644))

	_, err := ffmpeg.NewExtractor(1).ExtractFrames(context.Background(), "../../../test/fixtures/sample_1080p.mp4", filepath.Join(notADir, "frames"))
	assert.Error(t, err)
}

func TestExtractFramesBadGlobPatternReturnsError(t *testing.T) {
	// "[" em outDir quebra o pattern do filepath.Glob usado depois da extração
	// (o ffmpeg em si não liga pra esse caractere no path de saída).
	outDir := filepath.Join(t.TempDir(), "[bad")
	_, err := ffmpeg.NewExtractor(1).ExtractFrames(context.Background(), "../../../test/fixtures/sample_1080p.mp4", outDir)
	assert.Error(t, err)
}

func TestExtractFramesCreatesOutDir(t *testing.T) {
	parent := t.TempDir()
	outDir := filepath.Join(parent, "nested", "frames")
	_, err := os.Stat(outDir)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = ffmpeg.NewExtractor(1).ExtractFrames(context.Background(), "../../../test/fixtures/sample_1080p.mp4", outDir)
	require.NoError(t, err)

	info, err := os.Stat(outDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
