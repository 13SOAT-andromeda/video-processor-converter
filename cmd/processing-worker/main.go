package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	ddlambda "github.com/DataDog/dd-trace-go/contrib/aws/datadog-lambda-go/v2"
	awslambda "github.com/aws/aws-lambda-go/lambda"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/awsclient"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/config"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffmpeg"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ffprobe"
	lambdaadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/lambda"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/metrics"
	s3adapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/s3"
	sqsadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/sqs"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/ziparchive"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/process_video"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	statsd, err := metrics.New(cfg.StatsdAddr, logger)
	if err != nil {
		log.Fatalf("dogstatsd client: %v", err)
	}

	ctx := context.Background()
	awsCfg, err := awsclient.Load(ctx, cfg.Region)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	s3Client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
		if cfg.Endpoint != "" {
			o.UsePathStyle = true
		}
	})
	sqsClient := awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
	})

	uc := process_video.New(
		s3adapter.NewStorage(s3Client),
		ffprobe.NewProber(),
		ffmpeg.NewExtractor(cfg.FrameRate),
		ziparchive.NewArchiver(),
		sqsadapter.NewStatusPublisher(sqsClient, cfg.StatusQueueURL),
		statsd,
		process_video.Config{
			MaxWidth:  cfg.MaxWidth,
			MaxHeight: cfg.MaxHeight,
			TmpDir:    cfg.TmpDir,
		},
		logger,
	)

	handler := lambdaadapter.NewWorkerHandler(uc, statsd, logger)
	awslambda.Start(ddlambda.WrapFunction(handler.Handle, nil))
}
