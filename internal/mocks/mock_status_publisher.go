package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type MockStatusPublisher struct{ mock.Mock }

func (m *MockStatusPublisher) Publish(ctx context.Context, event domain.StatusEvent) error {
	return m.Called(ctx, event).Error(0)
}
