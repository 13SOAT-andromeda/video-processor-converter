package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

func TestNewProcessingJob(t *testing.T) {
	t.Run("chave raw válida deriva todos os campos", func(t *testing.T) {
		job, err := domain.NewProcessingJob("bucket", "lnk_123/raw/apresentacao.mp4")
		require.NoError(t, err)
		assert.Equal(t, "bucket", job.Bucket)
		assert.Equal(t, "lnk_123/raw/apresentacao.mp4", job.RawKey)
		assert.Equal(t, "lnk_123", job.LinkID)
		assert.Equal(t, "apresentacao.mp4", job.FileName)
		assert.Equal(t, "lnk_123/processed/apresentacao.zip", job.ProcessedKey)
	})

	t.Run("chave com subpastas extras usa o basename", func(t *testing.T) {
		job, err := domain.NewProcessingJob("bucket", "lnk_123/raw/sub/v.mov")
		require.NoError(t, err)
		assert.Equal(t, "v.mov", job.FileName)
		assert.Equal(t, "lnk_123/processed/v.zip", job.ProcessedKey)
	})

	t.Run("chave de processed retorna ErrNotRawKey", func(t *testing.T) {
		_, err := domain.NewProcessingJob("bucket", "lnk_123/processed/apresentacao.zip")
		assert.ErrorIs(t, err, domain.ErrNotRawKey)
	})

	t.Run("chave sem linkId retorna ErrNotRawKey", func(t *testing.T) {
		_, err := domain.NewProcessingJob("bucket", "raw/x.mp4")
		assert.ErrorIs(t, err, domain.ErrNotRawKey)
	})

	t.Run("chave vazia retorna ErrNotRawKey", func(t *testing.T) {
		_, err := domain.NewProcessingJob("bucket", "")
		assert.ErrorIs(t, err, domain.ErrNotRawKey)
	})

	t.Run("chave sem filename retorna ErrNotRawKey", func(t *testing.T) {
		_, err := domain.NewProcessingJob("bucket", "lnk_123/raw/")
		assert.ErrorIs(t, err, domain.ErrNotRawKey)
	})

	t.Run("linkId vazio retorna ErrNotRawKey", func(t *testing.T) {
		_, err := domain.NewProcessingJob("bucket", "/raw/x.mp4")
		assert.ErrorIs(t, err, domain.ErrNotRawKey)
	})
}

func TestResolutionFits(t *testing.T) {
	assert.True(t, domain.Resolution{Width: 1920, Height: 1080}.Fits(1920, 1080), "limite exato é aceito")
	assert.True(t, domain.Resolution{Width: 1280, Height: 720}.Fits(1920, 1080), "menor que o limite é aceito")
	assert.False(t, domain.Resolution{Width: 2560, Height: 1440}.Fits(1920, 1080), "acima do limite é rejeitado")
	assert.False(t, domain.Resolution{Width: 1920, Height: 1440}.Fits(1920, 1080), "basta uma dimensão exceder")
	assert.False(t, domain.Resolution{Width: 3840, Height: 1080}.Fits(1920, 1080), "basta uma dimensão exceder")
}
