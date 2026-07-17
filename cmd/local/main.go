// Package main é o invoker dev-only: faz long-polling numa fila do LocalStack e
// injeta cada mensagem no mesmo handler usado pela Lambda (sem ddlambda/lambda.Start).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

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
)

// handlerFunc é a assinatura comum aos dois handlers Lambda.
type handlerFunc func(context.Context, events.SQSEvent) (events.SQSEventResponse, error)

func main() {
	queueKind := flag.String("queue", "worker", "worker|dlq")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	awsCfg, err := awsclient.Load(ctx, cfg.Region)
	if err != nil {
		logger.Error("aws config", "err", err)
		os.Exit(1)
	}
	sqsClient := awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
	})

	var handle handlerFunc
	var queueURL string

	switch *queueKind {
	case "dlq":
		uc := handle_dlq.New(sqsadapter.NewStatusPublisher(sqsClient, cfg.StatusQueueURL), logger)
		handle = lambdaadapter.NewDLQHandler(uc, logger).Handle
		queueURL = cfg.DLQQueueURL
	default:
		s3Client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
			o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
			if cfg.Endpoint != "" {
				o.UsePathStyle = true
			}
		})
		uc := process_video.New(
			s3adapter.NewStorage(s3Client), ffprobe.NewProber(), ffmpeg.NewExtractor(cfg.FrameRate),
			ziparchive.NewArchiver(), sqsadapter.NewStatusPublisher(sqsClient, cfg.StatusQueueURL),
			process_video.Config{ExpectedWidth: cfg.ExpectedWidth, ExpectedHeight: cfg.ExpectedHeight, TmpDir: cfg.TmpDir},
			logger,
		)
		handle = lambdaadapter.NewWorkerHandler(uc, logger).Handle
		queueURL = cfg.WorkerQueueURL
	}

	logger.Info("local invoker started", "queue", *queueKind, "url", queueURL)
	poll(ctx, logger, sqsClient, queueURL, handle)
}

func poll(ctx context.Context, log *slog.Logger, client *awssqs.Client, queueURL string, handle handlerFunc) {
	for {
		if ctx.Err() != nil {
			return
		}
		out, err := client.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
			QueueUrl: aws.String(queueURL), MaxNumberOfMessages: 1, WaitTimeSeconds: 20,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("receive error", "err", err)
			time.Sleep(time.Second)
			continue
		}
		for _, m := range out.Messages {
			ev := events.SQSEvent{Records: []events.SQSMessage{{MessageId: aws.ToString(m.MessageId), Body: aws.ToString(m.Body)}}}
			resp, herr := handle(ctx, ev)
			// só deleta se NÃO foi reportado como falha (imita o ESM)
			failed := herr != nil || len(resp.BatchItemFailures) > 0
			if failed {
				log.Warn("message left for retry", "messageId", aws.ToString(m.MessageId))
				continue
			}
			if _, err := client.DeleteMessage(ctx, &awssqs.DeleteMessageInput{QueueUrl: aws.String(queueURL), ReceiptHandle: m.ReceiptHandle}); err != nil {
				log.Warn("delete failed", "err", err)
			}
		}
	}
}
