package ports

import (
	"context"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type VideoProber interface {
	Probe(ctx context.Context, path string) (domain.Resolution, error)
}
