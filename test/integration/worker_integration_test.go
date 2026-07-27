//go:build integration

package integration

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/awsclient"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/config"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffmpeg"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffprobe"
	lambdaadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/lambda"
	s3adapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/s3"
	sqsadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/sqs"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ziparchive"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/handle_dlq"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/process_video"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/mocks"
)

type testEnv struct {
	cfg       config.Config
	s3Client  *awss3.Client
	sqsClient *awssqs.Client
	storage   *s3adapter.Storage
	worker    *lambdaadapter.WorkerHandler
	dlq       *lambdaadapter.DLQHandler
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	cfg := config.Load()
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:4566"
	}

	awsCfg, err := awsclient.Load(context.Background(), cfg.Region)
	require.NoError(t, err)

	s3Client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
		o.UsePathStyle = true
	})
	sqsClient := awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	storage := s3adapter.NewStorage(s3Client)
	publisher := sqsadapter.NewStatusPublisher(sqsClient, cfg.StatusQueueURL)

	metrics := &mocks.MockMetrics{}
	uc := process_video.New(
		storage, ffprobe.NewProber(), ffmpeg.NewExtractor(cfg.FrameRate), ziparchive.NewArchiver(), publisher, metrics,
		process_video.Config{MaxWidth: cfg.MaxWidth, MaxHeight: cfg.MaxHeight, TmpDir: t.TempDir()},
		logger,
	)

	return &testEnv{
		cfg:       cfg,
		s3Client:  s3Client,
		sqsClient: sqsClient,
		storage:   storage,
		worker:    lambdaadapter.NewWorkerHandler(uc, metrics, logger),
		dlq:       lambdaadapter.NewDLQHandler(handle_dlq.New(publisher, metrics, logger), metrics, logger),
	}
}

func (e *testEnv) uploadFixture(t *testing.T, fixturePath, key string) {
	t.Helper()
	require.NoError(t, e.storage.Upload(context.Background(), e.cfg.Bucket, key, fixturePath, "video/mp4"))
	t.Cleanup(func() {
		_ = e.storage.Delete(context.Background(), e.cfg.Bucket, key)
	})
}

func (e *testEnv) sqsEventFor(t *testing.T, key string) events.SQSEvent {
	t.Helper()
	s3ev := events.S3Event{Records: []events.S3EventRecord{{
		S3: events.S3Entity{
			Bucket: events.S3Bucket{Name: e.cfg.Bucket},
			Object: events.S3Object{Key: key},
		},
	}}}
	body, err := json.Marshal(s3ev)
	require.NoError(t, err)
	return events.SQSEvent{Records: []events.SQSMessage{{MessageId: "it-msg", Body: string(body)}}}
}

// drainStatusQueue consome (e deleta) todas as mensagens visíveis da status-queue.
func (e *testEnv) drainStatusQueue(t *testing.T) []domain.StatusEvent {
	t.Helper()
	var out []domain.StatusEvent
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := e.sqsClient.ReceiveMessage(context.Background(), &awssqs.ReceiveMessageInput{
			QueueUrl: aws.String(e.cfg.StatusQueueURL), MaxNumberOfMessages: 10, WaitTimeSeconds: 1,
		})
		require.NoError(t, err)
		if len(resp.Messages) == 0 {
			break
		}
		for _, m := range resp.Messages {
			var ev domain.StatusEvent
			require.NoError(t, json.Unmarshal([]byte(aws.ToString(m.Body)), &ev))
			out = append(out, ev)
			_, err := e.sqsClient.DeleteMessage(context.Background(), &awssqs.DeleteMessageInput{
				QueueUrl: aws.String(e.cfg.StatusQueueURL), ReceiptHandle: m.ReceiptHandle,
			})
			require.NoError(t, err)
		}
	}
	return out
}

func statusesOf(evs []domain.StatusEvent) []string {
	out := make([]string, len(evs))
	for i, e := range evs {
		out[i] = e.Status
	}
	return out
}

func TestWorkerHappyPath(t *testing.T) {
	env := newTestEnv(t)
	env.drainStatusQueue(t) // limpa resíduos de execuções anteriores

	rawKey := "lnk_it/raw/sample_1080p.mp4"
	processedKey := "lnk_it/processed/sample_1080p.zip"
	env.uploadFixture(t, "../fixtures/sample_1080p.mp4", rawKey)
	t.Cleanup(func() { _ = env.storage.Delete(context.Background(), env.cfg.Bucket, processedKey) })

	resp, err := env.worker.Handle(context.Background(), env.sqsEventFor(t, rawKey))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)

	evs := env.drainStatusQueue(t)
	require.Equal(t, []string{domain.StatusProcessingStarted, domain.StatusProcessingCompleted}, statusesOf(evs))
	assert.Equal(t, "lnk_it", evs[0].LinkID)
	assert.Equal(t, processedKey, evs[1].S3ProcessedKey)

	// zip existe e contém frames; raw foi deletado
	exists, err := env.storage.Exists(context.Background(), env.cfg.Bucket, processedKey)
	require.NoError(t, err)
	assert.True(t, exists, "zip deve existir em processed/")

	zipPath := t.TempDir() + "/out.zip"
	require.NoError(t, env.storage.Download(context.Background(), env.cfg.Bucket, processedKey, zipPath))
	assertZipHasJPGs(t, zipPath)

	rawExists, err := env.storage.Exists(context.Background(), env.cfg.Bucket, rawKey)
	require.NoError(t, err)
	assert.False(t, rawExists, "raw deve ter sido deletado")
}

func TestWorkerInvalidResolution(t *testing.T) {
	env := newTestEnv(t)
	env.drainStatusQueue(t)

	rawKey := "lnk_it_bad/raw/sample_1440p.mp4"
	env.uploadFixture(t, "../fixtures/sample_1440p.mp4", rawKey)

	resp, err := env.worker.Handle(context.Background(), env.sqsEventFor(t, rawKey))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures, "resolução inválida é não-retryable")

	evs := env.drainStatusQueue(t)
	require.Equal(t, []string{domain.StatusProcessingStarted, domain.StatusProcessingFailed}, statusesOf(evs))
	assert.Equal(t, domain.ReasonInvalidResolution, evs[1].Reason)

	exists, err := env.storage.Exists(context.Background(), env.cfg.Bucket, "lnk_it_bad/processed/sample_1440p.zip")
	require.NoError(t, err)
	assert.False(t, exists, "não deve gerar zip")
}

// Resoluções menores que o máximo (1920x1080) também devem ser processadas.
func TestWorkerSmallerResolutionAllowed(t *testing.T) {
	env := newTestEnv(t)
	env.drainStatusQueue(t)

	rawKey := "lnk_it_720/raw/sample_720p.mp4"
	processedKey := "lnk_it_720/processed/sample_720p.zip"
	env.uploadFixture(t, "../fixtures/sample_720p.mp4", rawKey)
	t.Cleanup(func() { _ = env.storage.Delete(context.Background(), env.cfg.Bucket, processedKey) })

	resp, err := env.worker.Handle(context.Background(), env.sqsEventFor(t, rawKey))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)

	evs := env.drainStatusQueue(t)
	require.Equal(t, []string{domain.StatusProcessingStarted, domain.StatusProcessingCompleted}, statusesOf(evs))
	assert.Equal(t, processedKey, evs[1].S3ProcessedKey)
}

func TestWorkerIdempotency(t *testing.T) {
	env := newTestEnv(t)
	env.drainStatusQueue(t)

	rawKey := "lnk_it_idem/raw/sample_1080p.mp4"
	processedKey := "lnk_it_idem/processed/sample_1080p.zip"
	env.uploadFixture(t, "../fixtures/sample_1080p.mp4", rawKey)
	t.Cleanup(func() { _ = env.storage.Delete(context.Background(), env.cfg.Bucket, processedKey) })

	// 1ª execução: processa
	resp, err := env.worker.Handle(context.Background(), env.sqsEventFor(t, rawKey))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
	first := env.drainStatusQueue(t)
	require.Equal(t, []string{domain.StatusProcessingStarted, domain.StatusProcessingCompleted}, statusesOf(first))

	// 2ª execução (mesma mensagem): zip já existe → nada publicado, sem falha
	env.uploadFixture(t, "../fixtures/sample_1080p.mp4", rawKey) // simula reentrega com raw presente
	resp, err = env.worker.Handle(context.Background(), env.sqsEventFor(t, rawKey))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)

	second := env.drainStatusQueue(t)
	assert.Empty(t, second, "reentrega de vídeo já processado não deve publicar novos eventos")
}

func TestDLQHandlerPublishesMaxRetries(t *testing.T) {
	env := newTestEnv(t)
	env.drainStatusQueue(t)

	resp, err := env.dlq.Handle(context.Background(), env.sqsEventFor(t, "lnk_it/raw/x.mp4"))
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)

	evs := env.drainStatusQueue(t)
	require.Len(t, evs, 1)
	assert.Equal(t, domain.StatusProcessingFailed, evs[0].Status)
	assert.Equal(t, domain.ReasonMaxRetriesExceeded, evs[0].Reason)
	assert.Equal(t, "lnk_it", evs[0].LinkID)
}

func listZipEntries(zipPath string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names, nil
}

func assertZipHasJPGs(t *testing.T, zipPath string) {
	t.Helper()
	info, err := os.Stat(zipPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))

	names, err := listZipEntries(zipPath)
	require.NoError(t, err)
	jpgs := 0
	for _, n := range names {
		if len(n) > 4 && n[len(n)-4:] == ".jpg" {
			jpgs++
		}
	}
	assert.GreaterOrEqual(t, jpgs, 1, fmt.Sprintf("zip deve conter >=1 .jpg (entradas: %v)", names))
}
