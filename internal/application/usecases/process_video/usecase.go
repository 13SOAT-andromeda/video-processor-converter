package process_video

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type Config struct {
	ExpectedWidth  int
	ExpectedHeight int
	TmpDir         string
}

type UseCase struct {
	storage   ports.ObjectStorage
	prober    ports.VideoProber
	extractor ports.FrameExtractor
	archiver  ports.Archiver
	publisher ports.StatusPublisher
	cfg       Config
	log       *slog.Logger
}

func New(
	storage ports.ObjectStorage,
	prober ports.VideoProber,
	extractor ports.FrameExtractor,
	archiver ports.Archiver,
	publisher ports.StatusPublisher,
	cfg Config,
	log *slog.Logger,
) *UseCase {
	return &UseCase{storage, prober, extractor, archiver, publisher, cfg, log}
}

// Execute roda o pipeline para um job. Retorna:
//   - nil                        → concluído com sucesso (PROCESSING_COMPLETED publicado)
//   - domain.ErrAlreadyProcessed → zip já existia; nada publicado (idempotência)
//   - domain.ErrInvalidResolution→ PROCESSING_FAILED(invalid_resolution) publicado; não-retryable
//   - qualquer outro erro        → falha transitória (o handler deve pedir retry via BatchItemFailures)
func (uc *UseCase) Execute(ctx context.Context, job domain.ProcessingJob) error {
	log := uc.log.With("linkId", job.LinkID, "rawKey", job.RawKey)

	// (Passo 2 do spec) Idempotência: zip já existe?
	exists, err := uc.storage.Exists(ctx, job.Bucket, job.ProcessedKey)
	if err != nil {
		return fmt.Errorf("idempotency head: %w", err)
	}
	if exists {
		log.Info("already processed; skipping")
		return domain.ErrAlreadyProcessed
	}

	// (Passo 3) PROCESSING_STARTED
	if err := uc.publish(ctx, domain.StatusEvent{LinkID: job.LinkID, Status: domain.StatusProcessingStarted}); err != nil {
		return err
	}

	// Diretório de trabalho isolado por invocação
	workDir := filepath.Join(uc.cfg.TmpDir, "job-"+uuid.NewString())
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("mkdir workdir: %w", err)
	}
	defer os.RemoveAll(workDir) // limpa /tmp sempre

	inputPath := filepath.Join(workDir, job.FileName)
	framesDir := filepath.Join(workDir, "frames")
	zipPath := filepath.Join(workDir, "output.zip")

	// (Passo 4) Download
	if err := uc.storage.Download(ctx, job.Bucket, job.RawKey, inputPath); err != nil {
		return fmt.Errorf("download raw: %w", err)
	}

	// (Passo 5) Validação de resolução
	res, err := uc.prober.Probe(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	if !res.Equals(uc.cfg.ExpectedWidth, uc.cfg.ExpectedHeight) {
		log.Warn("invalid resolution", "width", res.Width, "height", res.Height)
		if err := uc.publish(ctx, domain.StatusEvent{
			LinkID: job.LinkID, Status: domain.StatusProcessingFailed, Reason: domain.ReasonInvalidResolution,
		}); err != nil {
			return err
		}
		return domain.ErrInvalidResolution
	}

	// (Passo 6) Extração de frames
	n, err := uc.extractor.ExtractFrames(ctx, inputPath, framesDir)
	if err != nil {
		return fmt.Errorf("extract frames: %w", err)
	}
	log.Info("frames extracted", "count", n)

	// (Passo 7) Zip
	if err := uc.archiver.Zip(ctx, framesDir, zipPath); err != nil {
		return fmt.Errorf("zip frames: %w", err)
	}

	// (Passo 8) Upload do zip
	if err := uc.storage.Upload(ctx, job.Bucket, job.ProcessedKey, zipPath, "application/zip"); err != nil {
		return fmt.Errorf("upload zip: %w", err)
	}

	// (Passo 9) Deleta o raw (best-effort: falha aqui não deve reprocessar tudo)
	if err := uc.storage.Delete(ctx, job.Bucket, job.RawKey); err != nil {
		log.Warn("failed to delete raw (continuing)", "err", err)
	}

	// (Passo 10) PROCESSING_COMPLETED
	if err := uc.publish(ctx, domain.StatusEvent{
		LinkID: job.LinkID, Status: domain.StatusProcessingCompleted, S3ProcessedKey: job.ProcessedKey,
	}); err != nil {
		return err
	}

	log.Info("processing completed")
	return nil
}

func (uc *UseCase) publish(ctx context.Context, e domain.StatusEvent) error {
	if err := uc.publisher.Publish(ctx, e); err != nil {
		return fmt.Errorf("publish %s: %w", e.Status, err)
	}
	return nil
}
