package handle_dlq_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/handle_dlq"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

func TestExecutePublishesMaxRetriesExceeded(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}
	publisher.On("Publish", mock.Anything, domain.StatusEvent{
		LinkID: "lnk_1",
		Status: domain.StatusProcessingFailed,
		Reason: domain.ReasonMaxRetriesExceeded,
	}).Return(nil)

	uc := handle_dlq.New(publisher, slog.New(slog.DiscardHandler))
	err := uc.Execute(context.Background(), "lnk_1")

	require.NoError(t, err)
	publisher.AssertExpectations(t)
}

func TestExecutePublishFailureReturnsWrappedError(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}
	sentinel := errors.New("sqs down")
	publisher.On("Publish", mock.Anything, mock.Anything).Return(sentinel)

	uc := handle_dlq.New(publisher, slog.New(slog.DiscardHandler))
	err := uc.Execute(context.Background(), "lnk_1")

	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
}
