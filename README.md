# video-processor-converter

Serviço `processing-worker` do desafio de processamento de vídeos (13SOAT / Andromeda). Consome eventos `ObjectCreated` do S3 via SQS (Lambda + Event Source Mapping), valida a resolução do vídeo (1920x1080), extrai 1 frame por segundo com ffmpeg, compacta os frames em um `.zip`, publica o resultado de volta no S3 e reporta o progresso na `video-processing-status-queue`. Inclui a Lambda companion `dlq-handler`, que marca como falha definitiva as mensagens que esgotaram as tentativas.

Spec funcional: [`context/specs/service-processing-worker.md`](context/specs/service-processing-worker.md) (diretório `context/` não é versionado).

## Como rodar local

Requisitos: Docker + Docker Compose, Go 1.25, ffmpeg/ffprobe no PATH.

```bash
make compose-up          # sobe LocalStack (S3+SQS) e cria bucket/filas/notificação
cp .env.example .env     # ajustar URLs se necessário
make run-local           # invoker local com long-polling na fila principal

# noutro terminal: subir um vídeo 1080p para disparar o pipeline
aws --endpoint-url=http://localhost:4566 s3 cp test/fixtures/sample_1080p.mp4 \
  s3://video-processor-bucket/lnk_demo/raw/sample_1080p.mp4

# conferir os eventos de status:
aws --endpoint-url=http://localhost:4566 sqs receive-message \
  --queue-url http://localhost:4566/000000000000/video-processing-status-queue --max-number-of-messages 10

# conferir o zip gerado:
aws --endpoint-url=http://localhost:4566 s3 ls s3://video-processor-bucket/lnk_demo/processed/
```

`make run-local-dlq` roda o invoker apontando para a DLQ (testa o `dlq-handler`).

## Build

```bash
make build          # compila os dois binários (linux/amd64)
make docker-build   # imagem de container do worker (ECR)
make dist-dlq       # gera dist/dlq-handler.zip (deploy via zip)
```

## Testes

```bash
make test               # unitários (rápidos, sem rede)
make test-cover         # unitários + cobertura
make test-integration   # integração (requer make compose-up antes)
```
