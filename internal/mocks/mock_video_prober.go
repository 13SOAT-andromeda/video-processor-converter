package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

type MockVideoProber struct{ mock.Mock }

func (m *MockVideoProber) Probe(ctx context.Context, path string) (domain.Resolution, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(domain.Resolution), args.Error(1)
}
