package config

import (
	"os"
	"strconv"
)

type Config struct {
	Region         string
	Endpoint       string // AWS_ENDPOINT_URL (vazio em produção)
	Bucket         string
	StatusQueueURL string
	// cmd/local:
	WorkerQueueURL string
	DLQQueueURL    string
	// pipeline:
	MaxWidth  int
	MaxHeight int
	FrameRate int
	TmpDir    string
}

func getEnv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		Region:         getEnv("AWS_REGION", "us-east-1"),
		Endpoint:       getEnv("AWS_ENDPOINT_URL", ""),
		Bucket:         getEnv("S3_BUCKET", ""),
		StatusQueueURL: getEnv("STATUS_QUEUE_URL", ""),
		WorkerQueueURL: getEnv("WORKER_QUEUE_URL", ""),
		DLQQueueURL:    getEnv("DLQ_QUEUE_URL", ""),
		MaxWidth:       getEnvInt("MAX_WIDTH", 1920),
		MaxHeight:      getEnvInt("MAX_HEIGHT", 1080),
		FrameRate:      getEnvInt("FRAME_RATE", 1),
		TmpDir:         getEnv("TMP_DIR", "/tmp"),
	}
}
