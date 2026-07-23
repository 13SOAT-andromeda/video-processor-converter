package lambda

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"

	"github.com/aws/aws-lambda-go/events"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/handle_dlq"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type DLQHandler struct {
	uc      *handle_dlq.UseCase
	metrics ports.Metrics
	log     *slog.Logger
}

func NewDLQHandler(uc *handle_dlq.UseCase, metrics ports.Metrics, log *slog.Logger) *DLQHandler {
	return &DLQHandler{uc: uc, metrics: metrics, log: log}
}

func (h *DLQHandler) Handle(ctx context.Context, ev events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure

	for _, rec := range ev.Records {
		if err := h.processRecord(ctx, rec); err != nil {
			h.log.Error("dlq record failed (will retry)", "messageId", rec.MessageId, "err", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: rec.MessageId})
		}
	}
	h.metrics.Count("batch.records", int64(len(ev.Records)))
	h.metrics.Count("batch.failures", int64(len(failures)))
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *DLQHandler) processRecord(ctx context.Context, rec events.SQSMessage) error {
	var s3ev events.S3Event
	if err := json.Unmarshal([]byte(rec.Body), &s3ev); err != nil {
		h.log.Error("malformed S3 event in DLQ; dropping", "messageId", rec.MessageId, "err", err)
		return nil
	}
	for _, r := range s3ev.Records {
		key, err := url.QueryUnescape(r.S3.Object.Key)
		if err != nil {
			h.log.Error("bad key encoding in DLQ; dropping", "key", r.S3.Object.Key)
			continue
		}
		job, err := domain.NewProcessingJob(r.S3.Bucket.Name, key)
		if err != nil {
			if errors.Is(err, domain.ErrNotRawKey) {
				h.log.Info("non-raw key in DLQ; skipping", "key", key)
				continue
			}
			return err
		}
		if err := h.uc.Execute(ctx, job.LinkID); err != nil {
			return err // transitório (ex.: SQS indisponível) → retry a partir da DLQ
		}
	}
	return nil
}
