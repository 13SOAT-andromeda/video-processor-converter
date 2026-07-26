variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment suffix used by the shared infra resource names"
  type        = string
  default     = "prod"
}

variable "image_tag" {
  description = "ECR image tag for the processing-worker (commit SHA pushed by deploy.yml)"
  type        = string
}

variable "dlq_zip_key" {
  description = "S3 key of the dlq-handler zip inside the artifacts bucket"
  type        = string
  default     = "dlq-handler/latest.zip"
}

variable "dd_api_key" {
  description = "Datadog API key"
  type        = string
  sensitive   = true
}
