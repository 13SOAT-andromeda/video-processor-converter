package domain

import (
	"errors"
	"path"
	"strings"
)

// Status transitions emitidos pelo worker (contrato da video-processing-status-queue).
const (
	StatusProcessingStarted   = "PROCESSING_STARTED"
	StatusProcessingCompleted = "PROCESSING_COMPLETED"
	StatusProcessingFailed    = "PROCESSING_FAILED"
)

// Reasons para PROCESSING_FAILED.
const (
	ReasonInvalidResolution  = "invalid_resolution"
	ReasonMaxRetriesExceeded = "max_retries_exceeded"
)

// Erros sentinela — casos NÃO-transitórios (o handler os trata como sucesso p/ o ESM não retentar).
var (
	ErrAlreadyProcessed  = errors.New("video already processed") // zip já existe no S3
	ErrInvalidResolution = errors.New("invalid video resolution")
	ErrNotRawKey         = errors.New("s3 key is not a raw upload") // proteção contra loop do processed/
)

// ProcessingJob descreve uma unidade de trabalho derivada de um evento S3.
type ProcessingJob struct {
	Bucket       string
	RawKey       string // {linkId}/raw/{fileName}
	LinkID       string
	FileName     string // basename com extensão (ex.: apresentacao.mp4)
	ProcessedKey string // {linkId}/processed/{base}.zip
}

// StatusEvent é o payload publicado na video-processing-status-queue.
// json:"eventType" (não "status"): contrato consumido pelo links-service
// (video-processor-link-api), que decodifica em app.StatusEvent.EventType —
// mesmo nome de campo usado por ele no LinkEvents/DynamoDB e no Makefile de
// simulação (make simulate-worker EVENT=...).
type StatusEvent struct {
	LinkID         string `json:"linkId"`
	Status         string `json:"eventType"`
	Reason         string `json:"reason,omitempty"`
	S3ProcessedKey string `json:"s3ProcessedKey,omitempty"`
}

// Resolution é o resultado do ffprobe.
type Resolution struct {
	Width  int
	Height int
}

// Fits diz se a resolução cabe no limite máximo permitido (ambas as dimensões).
func (r Resolution) Fits(maxW, maxH int) bool { return r.Width <= maxW && r.Height <= maxH }

// NewProcessingJob valida e deriva os campos a partir da chave S3 crua.
// Espera o layout {linkId}/raw/{fileName}. Qualquer outro layout → ErrNotRawKey
// (protege contra o próprio .zip em {linkId}/processed/ re-disparar o worker).
func NewProcessingJob(bucket, rawKey string) (ProcessingJob, error) {
	parts := strings.Split(rawKey, "/")
	if len(parts) < 3 || parts[1] != "raw" || parts[0] == "" || parts[len(parts)-1] == "" {
		return ProcessingJob{}, ErrNotRawKey
	}
	linkID := parts[0]
	fileName := parts[len(parts)-1]
	base := strings.TrimSuffix(fileName, path.Ext(fileName)) // apresentacao.mp4 -> apresentacao
	return ProcessingJob{
		Bucket:       bucket,
		RawKey:       rawKey,
		LinkID:       linkID,
		FileName:     fileName,
		ProcessedKey: linkID + "/processed/" + base + ".zip",
	}, nil
}
