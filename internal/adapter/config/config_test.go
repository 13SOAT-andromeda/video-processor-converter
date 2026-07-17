package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/config"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{
		"AWS_REGION", "AWS_ENDPOINT_URL", "S3_BUCKET", "STATUS_QUEUE_URL",
		"WORKER_QUEUE_URL", "DLQ_QUEUE_URL", "EXPECTED_WIDTH", "EXPECTED_HEIGHT",
		"FRAME_RATE", "TMP_DIR",
	} {
		t.Setenv(k, "")
	}

	cfg := config.Load()

	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Empty(t, cfg.Endpoint)
	assert.Empty(t, cfg.Bucket)
	assert.Equal(t, 1920, cfg.ExpectedWidth)
	assert.Equal(t, 1080, cfg.ExpectedHeight)
	assert.Equal(t, 1, cfg.FrameRate)
	assert.Equal(t, "/tmp", cfg.TmpDir)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "sa-east-1")
	t.Setenv("AWS_ENDPOINT_URL", "http://localhost:4566")
	t.Setenv("S3_BUCKET", "my-bucket")
	t.Setenv("STATUS_QUEUE_URL", "http://q/status")
	t.Setenv("WORKER_QUEUE_URL", "http://q/worker")
	t.Setenv("DLQ_QUEUE_URL", "http://q/dlq")
	t.Setenv("EXPECTED_WIDTH", "1280")
	t.Setenv("EXPECTED_HEIGHT", "720")
	t.Setenv("FRAME_RATE", "2")
	t.Setenv("TMP_DIR", "/var/tmp")

	cfg := config.Load()

	assert.Equal(t, "sa-east-1", cfg.Region)
	assert.Equal(t, "http://localhost:4566", cfg.Endpoint)
	assert.Equal(t, "my-bucket", cfg.Bucket)
	assert.Equal(t, "http://q/status", cfg.StatusQueueURL)
	assert.Equal(t, "http://q/worker", cfg.WorkerQueueURL)
	assert.Equal(t, "http://q/dlq", cfg.DLQQueueURL)
	assert.Equal(t, 1280, cfg.ExpectedWidth)
	assert.Equal(t, 720, cfg.ExpectedHeight)
	assert.Equal(t, 2, cfg.FrameRate)
	assert.Equal(t, "/var/tmp", cfg.TmpDir)
}

func TestLoadInvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("EXPECTED_WIDTH", "not-a-number")
	cfg := config.Load()
	assert.Equal(t, 1920, cfg.ExpectedWidth)
}
