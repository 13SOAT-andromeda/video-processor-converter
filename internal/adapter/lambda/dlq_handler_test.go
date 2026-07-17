package lambda_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	lambdaadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/lambda"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/handle_dlq"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

func newDLQHandler(publisher *mocks.MockStatusPublisher) *lambdaadapter.DLQHandler {
	logger := slog.New(slog.DiscardHandler)
	return lambdaadapter.NewDLQHandler(handle_dlq.New(publisher, logger), logger)
}

func TestDLQHandleExtractsLinkID(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}
	publisher.On("Publish", mock.Anything, domain.StatusEvent{
		LinkID: "lnk_123",
		Status: domain.StatusProcessingFailed,
		Reason: domain.ReasonMaxRetriesExceeded,
	}).Return(nil)

	resp, err := newDLQHandler(publisher).Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/apresentacao.mp4"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	publisher.AssertExpectations(t)
}

func TestDLQHandleNonRawKeySkips(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}

	resp, err := newDLQHandler(publisher).Handle(context.Background(), sqsEventForKey(t, "lnk_123/processed/x.zip"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestDLQHandlePublishFailureReportsRetry(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}
	publisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("sqs down"))

	resp, err := newDLQHandler(publisher).Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/apresentacao.mp4"))

	require.NoError(t, err)
	require.Len(t, resp.BatchItemFailures, 1)
	assert.Equal(t, "msg-1", resp.BatchItemFailures[0].ItemIdentifier)
}

func TestDLQHandleMalformedBodyIsDropped(t *testing.T) {
	publisher := &mocks.MockStatusPublisher{}

	ev := events.SQSEvent{Records: []events.SQSMessage{{MessageId: "msg-1", Body: "not-json"}}}
	resp, err := newDLQHandler(publisher).Handle(context.Background(), ev)

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
}
