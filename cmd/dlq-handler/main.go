package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	ddlambda "github.com/DataDog/dd-trace-go/contrib/aws/datadog-lambda-go/v2"
	awslambda "github.com/aws/aws-lambda-go/lambda"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/awsclient"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/config"
	lambdaadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/lambda"
	sqsadapter "github.com/13SOAT-andromeda/video-processor-converter/internal/adapter/sqs"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/usecases/handle_dlq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx := context.Background()
	awsCfg, err := awsclient.Load(ctx, cfg.Region)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	sqsClient := awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
		o.BaseEndpoint = awsclient.BaseEndpoint(cfg.Endpoint)
	})

	uc := handle_dlq.New(sqsadapter.NewStatusPublisher(sqsClient, cfg.StatusQueueURL), logger)
	handler := lambdaadapter.NewDLQHandler(uc, logger)

	awslambda.Start(ddlambda.WrapFunction(handler.Handle, nil))
}
