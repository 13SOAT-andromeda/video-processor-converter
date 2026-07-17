package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockObjectStorage struct{ mock.Mock }

func (m *MockObjectStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	args := m.Called(ctx, bucket, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockObjectStorage) Download(ctx context.Context, bucket, key, destPath string) error {
	return m.Called(ctx, bucket, key, destPath).Error(0)
}

func (m *MockObjectStorage) Upload(ctx context.Context, bucket, key, srcPath, contentType string) error {
	return m.Called(ctx, bucket, key, srcPath, contentType).Error(0)
}

func (m *MockObjectStorage) Delete(ctx context.Context, bucket, key string) error {
	return m.Called(ctx, bucket, key).Error(0)
}
