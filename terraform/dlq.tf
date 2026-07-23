# dlq-handler: só publica o status de falha na status queue, por isso o
# footprint mínimo (128MB/30s) e batch de até 10 mensagens.
resource "aws_lambda_function" "dlq_handler" {
  function_name = "video-processor-dlq-handler-${var.environment}"
  role          = data.aws_iam_role.lab_role.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"

  s3_bucket = data.aws_s3_bucket.artifacts.bucket
  s3_key    = var.dlq_zip_key

  timeout     = 30
  memory_size = 128

  # Datadog Lambda Extension: pro worker (Image) o binário vem embutido no
  # Dockerfile; em Lambda por zip a extension só existe via layer pública.
  # x86_64 pois o build usa GOARCH=amd64 (ver Dockerfile.dlq).
  layers = ["arn:aws:lambda:${var.region}:464622532012:layer:Datadog-Extension:98"]

  environment {
    variables = {
      STATUS_QUEUE_URL = data.aws_sqs_queue.video_processing_status.url

      DD_API_KEY       = var.dd_api_key
      DD_SITE          = "us5.datadoghq.com"
      DD_TRACE_ENABLED = "true"
      DD_ENV           = var.environment
      DD_SERVICE       = "dlq-handler"
      DD_VERSION       = trimsuffix(basename(var.dlq_zip_key), ".zip")
    }
  }
}

resource "aws_lambda_event_source_mapping" "dlq_handler" {
  event_source_arn = data.aws_sqs_queue.video_processing_dlq.arn
  function_name    = aws_lambda_function.dlq_handler.arn

  batch_size              = 10
  function_response_types = ["ReportBatchItemFailures"]
}
