package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"

	"github.com/aws/aws-lambda-go/events"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/process_video"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type WorkerHandler struct {
	uc      *process_video.UseCase
	metrics ports.Metrics
	log     *slog.Logger
}

func NewWorkerHandler(uc *process_video.UseCase, metrics ports.Metrics, log *slog.Logger) *WorkerHandler {
	return &WorkerHandler{uc: uc, metrics: metrics, log: log}
}

// Handle nunca retorna erro de função — falhas de item viram BatchItemFailures,
// então batch.failures é o único sinal de erro visível pro Datadog aqui
// (aws.lambda.errors nunca dispara nesse desenho).
func (h *WorkerHandler) Handle(ctx context.Context, ev events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure

	for _, rec := range ev.Records {
		if err := h.processRecord(ctx, rec); err != nil {
			h.log.Error("record failed (will retry)", "messageId", rec.MessageId, "err", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: rec.MessageId})
		}
	}
	h.metrics.Count("batch.records", int64(len(ev.Records)))
	h.metrics.Count("batch.failures", int64(len(failures)))
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

// processRecord retorna erro APENAS para falhas transitórias (que devem retentar).
// Casos não-retryable (chave não-raw, já processado, resolução inválida) retornam nil
// → o Event Source Mapping deleta a mensagem.
func (h *WorkerHandler) processRecord(ctx context.Context, rec events.SQSMessage) error {
	var s3ev events.S3Event
	if err := json.Unmarshal([]byte(rec.Body), &s3ev); err != nil {
		h.log.Error("malformed S3 event; dropping", "messageId", rec.MessageId, "err", err)
		return nil // malformado nunca vai processar; não retentar
	}

	for _, r := range s3ev.Records {
		bucket := r.S3.Bucket.Name
		key, err := url.QueryUnescape(r.S3.Object.Key) // S3 envia a key URL-encoded
		if err != nil {
			h.log.Error("bad object key encoding; dropping", "key", r.S3.Object.Key)
			continue
		}

		job, err := domain.NewProcessingJob(bucket, key)
		if err != nil {
			if errors.Is(err, domain.ErrNotRawKey) {
				h.log.Info("non-raw key; skipping", "key", key)
				continue // ex.: o próprio .zip em processed/ — não é erro
			}
			return err
		}

		err = h.uc.Execute(ctx, job)
		switch {
		case err == nil:
			h.metrics.Count("job.completed", 1)
			continue
		case errors.Is(err, domain.ErrAlreadyProcessed):
			h.metrics.Count("job.skipped", 1)
			continue
		case errors.Is(err, domain.ErrInvalidResolution):
			h.metrics.Count("job.rejected", 1, "reason:invalid_resolution")
			continue
		default:
			return err // transitório → retry
		}
	}
	return nil
}
