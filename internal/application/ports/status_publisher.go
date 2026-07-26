package ports

import (
	"context"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type StatusPublisher interface {
	Publish(ctx context.Context, event domain.StatusEvent) error
}
