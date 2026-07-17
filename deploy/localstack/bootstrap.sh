#!/usr/bin/env bash
set -euo pipefail

LS_URL="${AWS_ENDPOINT_URL:-http://localstack:4566}"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"
BUCKET="${S3_BUCKET:-video-processor-bucket}"
ACCOUNT="000000000000"

AWS_LS="aws --endpoint-url=$LS_URL --region=$REGION"

echo "Waiting for LocalStack..."
until $AWS_LS sqs list-queues >/dev/null 2>&1; do echo "  ...retry"; sleep 2; done

# ---- S3 bucket ----
echo "Creating bucket $BUCKET"
$AWS_LS s3api create-bucket --bucket "$BUCKET" 2>/dev/null || true

# ---- SQS queues ----
create_queue() { $AWS_LS sqs create-queue --queue-name "$1" --output text --query QueueUrl; }
queue_arn()   { $AWS_LS sqs get-queue-attributes --queue-url "$1" --attribute-names QueueArn --output text --query 'Attributes.QueueArn'; }

DLQ_URL=$(create_queue "video-processing-dlq")
DLQ_ARN=$(queue_arn "$DLQ_URL")

MAIN_URL=$(create_queue "video-processing-queue")
STATUS_URL=$(create_queue "video-processing-status-queue")
MAIN_ARN=$(queue_arn "$MAIN_URL")

# redrive principal -> DLQ (maxReceiveCount=3)
$AWS_LS sqs set-queue-attributes --queue-url "$MAIN_URL" --attributes \
  "{\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"$DLQ_ARN\\\",\\\"maxReceiveCount\\\":\\\"3\\\"}\",\"VisibilityTimeout\":\"1800\"}"

# permitir que o S3 publique na fila principal
$AWS_LS sqs set-queue-attributes --queue-url "$MAIN_URL" --attributes \
  "{\"Policy\":\"{\\\"Version\\\":\\\"2012-10-17\\\",\\\"Statement\\\":[{\\\"Effect\\\":\\\"Allow\\\",\\\"Principal\\\":\\\"*\\\",\\\"Action\\\":\\\"sqs:SendMessage\\\",\\\"Resource\\\":\\\"$MAIN_ARN\\\"}]}\"}"

# ---- S3 -> SQS notification ----
# Filtra por sufixo de vídeo p/ o .zip gerado em processed/ NÃO re-disparar o worker.
# (o worker também protege via ErrNotRawKey, mas o filtro reduz ruído local)
$AWS_LS s3api put-bucket-notification-configuration --bucket "$BUCKET" --notification-configuration "{
  \"QueueConfigurations\": [
    {\"QueueArn\":\"$MAIN_ARN\",\"Events\":[\"s3:ObjectCreated:*\"],\"Filter\":{\"Key\":{\"FilterRules\":[{\"Name\":\"suffix\",\"Value\":\".mp4\"}]}}}
  ]
}"

echo ""
echo "Bootstrap OK"
echo "  BUCKET            = $BUCKET"
echo "  WORKER_QUEUE_URL  = $MAIN_URL"
echo "  DLQ_QUEUE_URL     = $DLQ_URL"
echo "  STATUS_QUEUE_URL  = $STATUS_URL"
