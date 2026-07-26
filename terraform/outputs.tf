output "worker_function_name" {
  description = "Name of the processing-worker Lambda function"
  value       = aws_lambda_function.worker.function_name
}

output "worker_function_arn" {
  description = "ARN of the processing-worker Lambda function"
  value       = aws_lambda_function.worker.arn
}

output "dlq_handler_function_name" {
  description = "Name of the dlq-handler Lambda function"
  value       = aws_lambda_function.dlq_handler.function_name
}

output "dlq_handler_function_arn" {
  description = "ARN of the dlq-handler Lambda function"
  value       = aws_lambda_function.dlq_handler.arn
}
