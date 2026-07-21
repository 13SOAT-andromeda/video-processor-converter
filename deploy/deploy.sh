#!/usr/bin/env bash
set -euo pipefail

# Deploy manual do worker e/ou do dlq-handler direto da máquina local, para testes.
# Cobre o que o deploy.yml automatiza (build + upload/push) mais o terraform apply,
# que hoje não roda em nenhum workflow.
#
# Uso:
#   ./deploy/deploy.sh [target] [environment]
#
#   target:      worker | dlq | all   (default: all)
#   environment: nome usado pelo terraform var.environment (default: prod)
#
# Requer: aws cli autenticado com credenciais reais (não as de test/LocalStack
# do .env), docker e terraform instalados, e DD_API_KEY no ambiente.
#
# O terraform exige image_tag e dlq_zip_key sempre, mesmo deployando só um dos
# dois lambdas. Para o alvo que não está sendo redeployado, o script lê o valor
# atualmente publicado (via DD_VERSION, que espelha exatamente essas vars em
# worker.tf/dlq.tf) para não alterá-lo.

TARGET="${1:-all}"
ENVIRONMENT="${2:-prod}"
REGION="${AWS_REGION:-us-east-1}"
SHA="$(git rev-parse --short HEAD)"

case "$TARGET" in
  worker|dlq|all) ;;
  *)
    echo "Erro: target inválido '${TARGET}' (use: worker | dlq | all)" >&2
    exit 1
    ;;
esac

#if [[ -n "$(git status --porcelain)" ]]; then
#  echo "Aviso: há mudanças não commitadas -- o SHA ${SHA} usado no DD_VERSION" >&2
#  echo "não vai corresponder exatamente ao código deployado." >&2
#fi

if [[ -z "${DD_API_KEY:-}" ]]; then
  echo "Erro: defina DD_API_KEY no ambiente antes de rodar (ex: export DD_API_KEY=...)." >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
WORKER_FUNCTION="video-processor-worker-${ENVIRONMENT}"
DLQ_FUNCTION="video-processor-dlq-handler-${ENVIRONMENT}"
# account_id no sufixo: nome de bucket S3 é global e a conta do AWS Academy
# Lab reseta a cada sessão — mesmo motivo do bucket de state do Terraform
# (ver RUNBOOK.md do iac-video-processor-infra e terraform/data.tf deste repo).
ARTIFACTS_BUCKET="video-processor-artifacts-${ENVIRONMENT}-${ACCOUNT_ID}"

current_worker_image_tag() {
  aws lambda get-function-configuration --function-name "$WORKER_FUNCTION" \
    --query 'Environment.Variables.DD_VERSION' --output text
}

current_dlq_zip_key() {
  local version
  version="$(aws lambda get-function-configuration --function-name "$DLQ_FUNCTION" \
    --query 'Environment.Variables.DD_VERSION' --output text)"
  echo "dlq-handler/${version}.zip"
}

deploy_worker() {
  echo "==> Build & push da imagem do worker (SHA ${SHA})"
  local repo_uri
  repo_uri="$(aws ecr describe-repositories --repository-names "video-processor-worker-${ENVIRONMENT}" \
    --query 'repositories[0].repositoryUri' --output text)"

  aws ecr get-login-password --region "$REGION" \
    | docker login --username AWS --password-stdin "${repo_uri%%/*}"

  # --provenance=false --sbom=false evita que o buildx anexe manifests de
  # attestation à imagem (default do buildx/containerd image store). O Lambda
  # rejeita esse formato multi-manifest no CreateFunction com "Source image
  # ... is not valid", mesmo a imagem funcionando normal via docker pull/run.
  docker buildx build --platform linux/amd64 --provenance=false --sbom=false \
    -t "${repo_uri}:${SHA}" -t "${repo_uri}:latest" \
    --push "$REPO_ROOT"
}

deploy_dlq() {
  echo "==> Build do dlq-handler.zip (Dockerfile.dlq)"
  rm -rf "${REPO_ROOT}/dist"
  docker build -f "${REPO_ROOT}/Dockerfile.dlq" --target artifact --output type=local,dest="${REPO_ROOT}/dist" "$REPO_ROOT"

  echo "==> Upload pro S3 (s3://${ARTIFACTS_BUCKET}/dlq-handler/)"
  aws s3 cp "${REPO_ROOT}/dist/dlq-handler.zip" "s3://${ARTIFACTS_BUCKET}/dlq-handler/${SHA}.zip"
  aws s3 cp "${REPO_ROOT}/dist/dlq-handler.zip" "s3://${ARTIFACTS_BUCKET}/dlq-handler/latest.zip"
}

if [[ "$TARGET" == "worker" || "$TARGET" == "all" ]]; then
  deploy_worker
  IMAGE_TAG="$SHA"
else
  echo "==> Mantendo image_tag atual do worker (não incluso neste deploy)"
  IMAGE_TAG="$(current_worker_image_tag)"
fi

if [[ "$TARGET" == "dlq" || "$TARGET" == "all" ]]; then
  deploy_dlq
  DLQ_ZIP_KEY="dlq-handler/${SHA}.zip"
else
  echo "==> Mantendo dlq_zip_key atual (não incluso neste deploy)"
  DLQ_ZIP_KEY="$(current_dlq_zip_key)"
fi

echo "==> terraform apply"
echo "    environment  = ${ENVIRONMENT}"
echo "    image_tag    = ${IMAGE_TAG}"
echo "    dlq_zip_key  = ${DLQ_ZIP_KEY}"

cd "${REPO_ROOT}/terraform"
terraform init -input=false
terraform apply \
  -var="environment=${ENVIRONMENT}" \
  -var="image_tag=${IMAGE_TAG}" \
  -var="dlq_zip_key=${DLQ_ZIP_KEY}" \
  -var="dd_api_key=${DD_API_KEY}"
