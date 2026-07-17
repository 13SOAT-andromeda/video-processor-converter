package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockArchiver struct{ mock.Mock }

func (m *MockArchiver) Zip(ctx context.Context, srcDir, destZipPath string) error {
	return m.Called(ctx, srcDir, destZipPath).Error(0)
}
