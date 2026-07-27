resource "aws_lambda_function" "worker" {
  function_name = "video-processor-worker-${var.environment}"
  role          = data.aws_iam_role.lab_role.arn
  package_type  = "Image"
  image_uri     = "${data.aws_ecr_repository.worker.repository_url}:${var.image_tag}"

  timeout     = 300
  memory_size = 2048

  ephemeral_storage {
    size = 2048
  }

  environment {
    variables = {
      S3_BUCKET        = data.aws_s3_bucket.videos.bucket
      STATUS_QUEUE_URL = data.aws_sqs_queue.video_processing_status.url
      MAX_WIDTH        = "1920"
      MAX_HEIGHT       = "1080"
      FRAME_RATE       = "1"
      TMP_DIR          = "/tmp"

      DD_API_KEY       = var.dd_api_key
      DD_SITE          = "us5.datadoghq.com"
      DD_TRACE_ENABLED = "true"
      DD_ENV           = var.environment
      DD_SERVICE       = "processing-worker"
      DD_VERSION       = var.image_tag
    }
  }
}

resource "aws_lambda_event_source_mapping" "worker" {
  event_source_arn = data.aws_sqs_queue.video_processing.arn
  function_name    = aws_lambda_function.worker.arn

  batch_size              = 1
  function_response_types = ["ReportBatchItemFailures"]

  scaling_config {
    maximum_concurrency = 10
  }
}
