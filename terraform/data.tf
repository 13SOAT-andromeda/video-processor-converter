# Recursos compartilhados provisionados pelo iac-video-processor-infra
# (filas, buckets e ECR) — consumidos por data source para não acoplar states.

data "aws_ecr_repository" "worker" {
  name = "video-processor-worker-${var.environment}"
}

data "aws_sqs_queue" "video_processing" {
  name = "video-processing-queue-${var.environment}"
}

data "aws_sqs_queue" "video_processing_dlq" {
  name = "video-processing-dlq-${var.environment}"
}

data "aws_sqs_queue" "video_processing_status" {
  name = "video-processing-status-queue-${var.environment}"
}

data "aws_s3_bucket" "videos" {
  bucket = "video-processor-bucket-${var.environment}"
}

data "aws_s3_bucket" "artifacts" {
  bucket = "video-processor-artifacts-${var.environment}"
}
