# ---- build ----
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bootstrap ./cmd/processing-worker

# ---- runtime (custom runtime provided.al2023) ----
FROM public.ecr.aws/lambda/provided:al2023

# ffmpeg/ffprobe estáticos x86_64 (imagem pinnável — sem curl no build)
COPY --from=mwader/static-ffmpeg:7.1 /ffmpeg  /usr/local/bin/ffmpeg
COPY --from=mwader/static-ffmpeg:7.1 /ffprobe /usr/local/bin/ffprobe

# binário da função
COPY --from=builder /bootstrap /var/runtime/bootstrap

# Datadog Lambda Extension (multi-arch; resolve amd64)
COPY --from=public.ecr.aws/datadog/lambda-extension:latest /opt/extensions/datadog-agent /opt/extensions/datadog-agent

CMD ["bootstrap"]
