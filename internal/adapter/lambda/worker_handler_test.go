package lambda_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	lambdaadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/lambda"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/process_video"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

type workerFixture struct {
	storage   *mocks.MockObjectStorage
	prober    *mocks.MockVideoProber
	extractor *mocks.MockFrameExtractor
	archiver  *mocks.MockArchiver
	publisher *mocks.MockStatusPublisher
	handler   *lambdaadapter.WorkerHandler
}

func newWorkerFixture(t *testing.T) *workerFixture {
	t.Helper()
	f := &workerFixture{
		storage:   &mocks.MockObjectStorage{},
		prober:    &mocks.MockVideoProber{},
		extractor: &mocks.MockFrameExtractor{},
		archiver:  &mocks.MockArchiver{},
		publisher: &mocks.MockStatusPublisher{},
	}
	logger := slog.New(slog.DiscardHandler)
	uc := process_video.New(
		f.storage, f.prober, f.extractor, f.archiver, f.publisher,
		process_video.Config{MaxWidth: 1920, MaxHeight: 1080, TmpDir: t.TempDir()},
		logger,
	)
	f.handler = lambdaadapter.NewWorkerHandler(uc, logger)
	return f
}

func sqsEventForKey(t *testing.T, key string) events.SQSEvent {
	t.Helper()
	s3ev := events.S3Event{Records: []events.S3EventRecord{{
		S3: events.S3Entity{
			Bucket: events.S3Bucket{Name: "bucket"},
			Object: events.S3Object{Key: key},
		},
	}}}
	body, err := json.Marshal(s3ev)
	require.NoError(t, err)
	return events.SQSEvent{Records: []events.SQSMessage{{MessageId: "msg-1", Body: string(body)}}}
}

func TestHandleNonRawKeySkips(t *testing.T) {
	f := newWorkerFixture(t)

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/processed/apresentacao.zip"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	f.storage.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleTransientErrorReportsFailure(t *testing.T) {
	f := newWorkerFixture(t)
	f.storage.On("Exists", mock.Anything, "bucket", mock.Anything).Return(false, errors.New("s3 down"))

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/apresentacao.mp4"))

	require.NoError(t, err)
	require.Len(t, resp.BatchItemFailures, 1)
	assert.Equal(t, "msg-1", resp.BatchItemFailures[0].ItemIdentifier)
}

func TestHandleInvalidResolutionIsNotFailure(t *testing.T) {
	f := newWorkerFixture(t)
	f.storage.On("Exists", mock.Anything, "bucket", mock.Anything).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, mock.Anything).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", mock.Anything, mock.Anything).Return(nil)
	f.prober.On("Probe", mock.Anything, mock.Anything).Return(domain.Resolution{Width: 2560, Height: 1440}, nil)

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/apresentacao.mp4"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
}

func TestHandleAlreadyProcessedIsNotFailure(t *testing.T) {
	f := newWorkerFixture(t)
	f.storage.On("Exists", mock.Anything, "bucket", mock.Anything).Return(true, nil)

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/apresentacao.mp4"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
}

func TestHandleMalformedBodyIsDropped(t *testing.T) {
	f := newWorkerFixture(t)

	ev := events.SQSEvent{Records: []events.SQSMessage{{MessageId: "msg-1", Body: "not-json"}}}
	resp, err := f.handler.Handle(context.Background(), ev)

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
}

func TestHandleBadKeyEncodingIsDropped(t *testing.T) {
	f := newWorkerFixture(t)

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/%zzfile.mp4"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	f.storage.AssertNotCalled(t, "Exists", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleURLEncodedKeyIsUnescaped(t *testing.T) {
	f := newWorkerFixture(t)
	// "lnk_123/raw/video+final.mp4" chega URL-encoded; QueryUnescape converte '+' em espaço
	f.storage.On("Exists", mock.Anything, "bucket", "lnk_123/processed/video final.zip").Return(true, nil)

	resp, err := f.handler.Handle(context.Background(), sqsEventForKey(t, "lnk_123/raw/video+final.mp4"))

	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	f.storage.AssertExpectations(t)
}
