package process_video_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/process_video"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

type fixture struct {
	storage   *mocks.MockObjectStorage
	prober    *mocks.MockVideoProber
	extractor *mocks.MockFrameExtractor
	archiver  *mocks.MockArchiver
	publisher *mocks.MockStatusPublisher
	uc        *process_video.UseCase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		storage:   &mocks.MockObjectStorage{},
		prober:    &mocks.MockVideoProber{},
		extractor: &mocks.MockFrameExtractor{},
		archiver:  &mocks.MockArchiver{},
		publisher: &mocks.MockStatusPublisher{},
	}
	f.uc = process_video.New(
		f.storage, f.prober, f.extractor, f.archiver, f.publisher,
		process_video.Config{MaxWidth: 1920, MaxHeight: 1080, TmpDir: t.TempDir()},
		slog.New(slog.DiscardHandler),
	)
	return f
}

func newJob(t *testing.T) domain.ProcessingJob {
	t.Helper()
	job, err := domain.NewProcessingJob("bucket", "lnk_123/raw/apresentacao.mp4")
	require.NoError(t, err)
	return job
}

func statusEvent(status string) any {
	return mock.MatchedBy(func(e domain.StatusEvent) bool { return e.Status == status })
}

func TestExecuteAlreadyProcessed(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(true, nil)

	err := f.uc.Execute(context.Background(), job)

	assert.ErrorIs(t, err, domain.ErrAlreadyProcessed)
	f.publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestExecuteInvalidResolution(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", job.RawKey, mock.Anything).Return(nil)
	f.prober.On("Probe", mock.Anything, mock.Anything).Return(domain.Resolution{Width: 2560, Height: 1440}, nil)
	f.publisher.On("Publish", mock.Anything, mock.MatchedBy(func(e domain.StatusEvent) bool {
		return e.Status == domain.StatusProcessingFailed && e.Reason == domain.ReasonInvalidResolution
	})).Return(nil)

	err := f.uc.Execute(context.Background(), job)

	assert.ErrorIs(t, err, domain.ErrInvalidResolution)
	f.extractor.AssertNotCalled(t, "ExtractFrames", mock.Anything, mock.Anything, mock.Anything)
	f.publisher.AssertExpectations(t)
}

func TestExecuteHappyPath(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", job.RawKey, mock.Anything).Return(nil)
	f.prober.On("Probe", mock.Anything, mock.Anything).Return(domain.Resolution{Width: 1920, Height: 1080}, nil)
	f.extractor.On("ExtractFrames", mock.Anything, mock.Anything, mock.Anything).Return(10, nil)
	f.archiver.On("Zip", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.storage.On("Upload", mock.Anything, "bucket", job.ProcessedKey, mock.Anything, "application/zip").Return(nil)
	f.storage.On("Delete", mock.Anything, "bucket", job.RawKey).Return(nil)
	f.publisher.On("Publish", mock.Anything, mock.MatchedBy(func(e domain.StatusEvent) bool {
		return e.Status == domain.StatusProcessingCompleted &&
			e.S3ProcessedKey == "lnk_123/processed/apresentacao.zip"
	})).Return(nil)

	err := f.uc.Execute(context.Background(), job)

	require.NoError(t, err)
	f.storage.AssertExpectations(t)
	f.publisher.AssertExpectations(t)
	f.extractor.AssertExpectations(t)
	f.archiver.AssertExpectations(t)
}

func TestExecuteSmallerResolutionAllowed(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", job.RawKey, mock.Anything).Return(nil)
	f.prober.On("Probe", mock.Anything, mock.Anything).Return(domain.Resolution{Width: 1280, Height: 720}, nil)
	f.extractor.On("ExtractFrames", mock.Anything, mock.Anything, mock.Anything).Return(10, nil)
	f.archiver.On("Zip", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.storage.On("Upload", mock.Anything, "bucket", job.ProcessedKey, mock.Anything, "application/zip").Return(nil)
	f.storage.On("Delete", mock.Anything, "bucket", job.RawKey).Return(nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingCompleted)).Return(nil)

	err := f.uc.Execute(context.Background(), job)

	require.NoError(t, err)
	f.publisher.AssertExpectations(t)
	f.extractor.AssertExpectations(t)
}

func TestExecuteTransientDownloadFailure(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", job.RawKey, mock.Anything).Return(errors.New("s3 down"))

	err := f.uc.Execute(context.Background(), job)

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrAlreadyProcessed)
	assert.NotErrorIs(t, err, domain.ErrInvalidResolution)
	f.publisher.AssertNotCalled(t, "Publish", mock.Anything, statusEvent(domain.StatusProcessingCompleted))
}

func TestExecutePublishStartedFailure(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(errors.New("sqs down"))

	err := f.uc.Execute(context.Background(), job)

	require.Error(t, err)
	f.storage.AssertNotCalled(t, "Download", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestExecuteDeleteRawFailureStillCompletes(t *testing.T) {
	f := newFixture(t)
	job := newJob(t)

	f.storage.On("Exists", mock.Anything, "bucket", job.ProcessedKey).Return(false, nil)
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingStarted)).Return(nil)
	f.storage.On("Download", mock.Anything, "bucket", job.RawKey, mock.Anything).Return(nil)
	f.prober.On("Probe", mock.Anything, mock.Anything).Return(domain.Resolution{Width: 1920, Height: 1080}, nil)
	f.extractor.On("ExtractFrames", mock.Anything, mock.Anything, mock.Anything).Return(5, nil)
	f.archiver.On("Zip", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.storage.On("Upload", mock.Anything, "bucket", job.ProcessedKey, mock.Anything, "application/zip").Return(nil)
	f.storage.On("Delete", mock.Anything, "bucket", job.RawKey).Return(errors.New("delete denied"))
	f.publisher.On("Publish", mock.Anything, statusEvent(domain.StatusProcessingCompleted)).Return(nil)

	err := f.uc.Execute(context.Background(), job)

	require.NoError(t, err)
	f.publisher.AssertExpectations(t)
}
