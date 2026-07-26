-include .env
export

.PHONY: help tidy build build-worker build-dlq dist-dlq test test-cover test-integration lint vet \
        docker-build compose-up compose-down run-local run-local-dlq clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "\033[36m%-16s\033[0m %s\n",$$1,$$2}'

tidy: ## go mod tidy
	go mod tidy

build: build-worker build-dlq ## Compila os dois binários localmente

build-worker: ## Compila o processing-worker
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/processing-worker ./cmd/processing-worker

build-dlq: ## Compila o dlq-handler (bootstrap para zip)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/bootstrap ./cmd/dlq-handler

dist-dlq: ## Gera dist/dlq-handler.zip
	docker build -f Dockerfile.dlq --target artifact --output type=local,dest=dist .

test: ## Testes unitários
	go test ./...

test-cover: ## Testes unitários com cobertura
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1

test-integration: ## Testes de integração (requer LocalStack no ar)
	go test -tags=integration ./test/... -v

vet: ## go vet
	go vet ./...

lint: ## golangci-lint (precisa estar instalado)
	golangci-lint run

docker-build: ## Build da imagem do worker
	docker build -t video-processor-worker:local .

compose-up: ## Sobe LocalStack (s3, sqs) + bootstrap
	docker compose up -d --build

compose-down: ## Derruba o ambiente local
	docker compose down -v

run-local: ## Roda o invoker local apontando para a fila principal
	go run ./cmd/local -queue worker

run-local-dlq: ## Roda o invoker local apontando para a DLQ
	go run ./cmd/local -queue dlq

clean: ## Remove artefatos
	rm -rf bin dist coverage.out coverage.html
