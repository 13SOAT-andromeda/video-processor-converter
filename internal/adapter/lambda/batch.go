package lambda

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
)

// processBatch roda process em cada registro do lote, conta registros/falhas
// via metrics e monta a resposta de batch item failures -- lógica compartilhada
// entre WorkerHandler.Handle e DLQHandler.Handle.
func processBatch(
	ctx context.Context,
	records []events.SQSMessage,
	metrics ports.Metrics,
	log *slog.Logger,
	failureMsg string,
	process func(context.Context, events.SQSMessage) error,
) events.SQSEventResponse {
	var failures []events.SQSBatchItemFailure
	for _, rec := range records {
		if err := process(ctx, rec); err != nil {
			log.Error(failureMsg, "messageId", rec.MessageId, "err", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: rec.MessageId})
		}
	}
	metrics.Count("batch.records", int64(len(records)))
	metrics.Count("batch.failures", int64(len(failures)))
	return events.SQSEventResponse{BatchItemFailures: failures}
}
