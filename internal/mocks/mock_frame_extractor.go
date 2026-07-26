package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockFrameExtractor struct{ mock.Mock }

func (m *MockFrameExtractor) ExtractFrames(ctx context.Context, inputPath, outDir string) (int, error) {
	args := m.Called(ctx, inputPath, outDir)
	return args.Int(0), args.Error(1)
}
