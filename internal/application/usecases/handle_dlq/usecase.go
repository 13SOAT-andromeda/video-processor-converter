package handle_dlq

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type UseCase struct {
	publisher ports.StatusPublisher
	log       *slog.Logger
}

func New(publisher ports.StatusPublisher, log *slog.Logger) *UseCase {
	return &UseCase{publisher: publisher, log: log}
}

func (uc *UseCase) Execute(ctx context.Context, linkID string) error {
	uc.log.Warn("processing exhausted retries; marking failed", "linkId", linkID)
	if err := uc.publisher.Publish(ctx, domain.StatusEvent{
		LinkID: linkID,
		Status: domain.StatusProcessingFailed,
		Reason: domain.ReasonMaxRetriesExceeded,
	}); err != nil {
		return fmt.Errorf("publish failed status: %w", err)
	}
	return nil
}
