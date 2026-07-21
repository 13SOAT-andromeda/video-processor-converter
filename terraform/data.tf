# Recursos compartilhados provisionados pelo iac-video-processor-infra
# (filas, buckets e ECR) — consumidos por data source para não acoplar states.

# Buckets levam o account_id no nome (mesmo motivo do bucket de state do
# Terraform, ver RUNBOOK.md do iac-video-processor-infra): nome de bucket S3
# é global, e a conta do AWS Academy Lab reseta a cada sessão — sem o sufixo,
# "video-processor-bucket-prod" colide com o mesmo nome já usado por outra
# conta.
data "aws_caller_identity" "current" {}

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
  bucket = "video-processor-bucket-${var.environment}-${data.aws_caller_identity.current.account_id}"
}

data "aws_s3_bucket" "artifacts" {
  bucket = "video-processor-artifacts-${var.environment}-${data.aws_caller_identity.current.account_id}"
}
