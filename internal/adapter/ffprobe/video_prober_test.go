package ffprobe_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffprobe"
)

// Usa o ffprobe real (instalado no runner de CI e localmente) contra os
// fixtures em test/fixtures — não é teste de integração: não toca AWS/LocalStack.

func TestProbeReturnsResolution(t *testing.T) {
	res, err := ffprobe.NewProber().Probe(context.Background(), "../../../test/fixtures/sample_1080p.mp4")
	require.NoError(t, err)
	assert.Equal(t, 1920, res.Width)
	assert.Equal(t, 1080, res.Height)
}

func TestProbeInvalidVideoReturnsError(t *testing.T) {
	_, err := ffprobe.NewProber().Probe(context.Background(), "../../../test/fixtures/invalid_video.mp4")
	assert.Error(t, err)
}

func TestProbeMissingFileReturnsError(t *testing.T) {
	_, err := ffprobe.NewProber().Probe(context.Background(), "../../../test/fixtures/does_not_exist.mp4")
	assert.Error(t, err)
}
