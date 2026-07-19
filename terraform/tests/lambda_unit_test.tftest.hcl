mock_provider "aws" {
  mock_data "aws_iam_role" {
    defaults = {
      arn  = "arn:aws:iam::123456789012:role/LabRole"
      name = "LabRole"
    }
  }

  mock_data "aws_ecr_repository" {
    defaults = {
      repository_url = "123456789012.dkr.ecr.us-east-1.amazonaws.com/video-processor-worker-prod"
    }
  }
}

variables {
  image_tag  = "abc1234"
  dd_api_key = "test-key"
}

run "worker_sized_and_wired_per_spec" {
  command = plan

  assert {
    condition     = aws_lambda_function.worker.function_name == "video-processor-worker-prod"
    error_message = "Expected the worker function name to follow the video-processor-worker-${var.environment} convention"
  }

  assert {
    condition     = aws_lambda_function.worker.image_uri == "123456789012.dkr.ecr.us-east-1.amazonaws.com/video-processor-worker-prod:abc1234"
    error_message = "Expected the worker image_uri to be the shared ECR repository URL pinned at var.image_tag"
  }

  assert {
    condition     = aws_lambda_function.worker.timeout == 300
    error_message = "Expected a 300s timeout — 1/6 of the queue's 1800s visibility timeout, per the AWS ESM sizing rule"
  }

  assert {
    condition     = aws_lambda_function.worker.memory_size == 2048
    error_message = "Expected 2048MB of memory (~1.2 vCPU for ffmpeg)"
  }

  assert {
    condition     = aws_lambda_function.worker.ephemeral_storage[0].size == 2048
    error_message = "Expected 2048MB of ephemeral storage — /tmp holds the downloaded video, frames and zip"
  }

  assert {
    condition     = aws_lambda_function.worker.environment[0].variables["STATUS_QUEUE_URL"] != ""
    error_message = "Expected STATUS_QUEUE_URL to be wired from the shared status queue data source"
  }

  assert {
    condition     = contains(aws_lambda_event_source_mapping.worker.function_response_types, "ReportBatchItemFailures")
    error_message = "Expected ReportBatchItemFailures on the worker ESM — without it the handler's batchItemFailures response is ignored"
  }

  assert {
    condition     = aws_lambda_event_source_mapping.worker.batch_size == 1
    error_message = "Expected batch_size 1 — one ffmpeg run per invocation; a batch would not fit the 300s timeout"
  }

  assert {
    condition     = aws_lambda_event_source_mapping.worker.scaling_config[0].maximum_concurrency == 2
    error_message = "Expected maximum_concurrency 2 as the Academy cost guard"
  }
}

run "dlq_handler_zip_runtime_and_esm" {
  command = plan

  assert {
    condition     = aws_lambda_function.dlq_handler.function_name == "video-processor-dlq-handler-prod"
    error_message = "Expected the dlq-handler function name to follow the video-processor-dlq-handler-${var.environment} convention"
  }

  assert {
    condition     = aws_lambda_function.dlq_handler.runtime == "provided.al2023" && aws_lambda_function.dlq_handler.handler == "bootstrap"
    error_message = "Expected the dlq-handler to run the Go custom runtime (provided.al2023, bootstrap handler)"
  }

  assert {
    condition     = aws_lambda_function.dlq_handler.s3_key == "dlq-handler/latest.zip"
    error_message = "Expected the default zip key to match what the converter CD publishes (dlq-handler/latest.zip)"
  }

  assert {
    condition     = aws_lambda_function.dlq_handler.memory_size == 128 && aws_lambda_function.dlq_handler.timeout == 30
    error_message = "Expected the minimal 128MB/30s footprint — the handler only publishes a status message"
  }

  assert {
    condition     = contains(aws_lambda_event_source_mapping.dlq_handler.function_response_types, "ReportBatchItemFailures")
    error_message = "Expected ReportBatchItemFailures on the dlq-handler ESM"
  }

  assert {
    condition     = aws_lambda_event_source_mapping.dlq_handler.batch_size == 10
    error_message = "Expected batch_size 10 on the DLQ ESM — the handler is light enough for full SQS batches"
  }
}
