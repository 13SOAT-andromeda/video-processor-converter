package domain

import (
	"errors"
	"path"
	"strings"
)

const (
	StatusProcessingStarted   = "PROCESSING_STARTED"
	StatusProcessingCompleted = "PROCESSING_COMPLETED"
	StatusProcessingFailed    = "PROCESSING_FAILED"
)

const (
	ReasonInvalidResolution  = "invalid_resolution"
	ReasonMaxRetriesExceeded = "max_retries_exceeded"
)

var (
	ErrAlreadyProcessed  = errors.New("video already processed")
	ErrInvalidResolution = errors.New("invalid video resolution")
	ErrNotRawKey         = errors.New("s3 key is not a raw upload")
)

type ProcessingJob struct {
	Bucket       string
	RawKey       string // {linkId}/raw/{fileName}
	LinkID       string
	FileName     string
	ProcessedKey string
}

type StatusEvent struct {
	LinkID         string `json:"linkId"`
	Status         string `json:"eventType"`
	Reason         string `json:"reason,omitempty"`
	S3ProcessedKey string `json:"s3ProcessedKey,omitempty"`
}
type Resolution struct {
	Width  int
	Height int
}

func (r Resolution) Fits(maxW, maxH int) bool { return r.Width <= maxW && r.Height <= maxH }

func NewProcessingJob(bucket, rawKey string) (ProcessingJob, error) {
	parts := strings.Split(rawKey, "/")
	if len(parts) < 3 || parts[1] != "raw" || parts[0] == "" || parts[len(parts)-1] == "" {
		return ProcessingJob{}, ErrNotRawKey
	}
	linkID := parts[0]
	fileName := parts[len(parts)-1]
	if fileName == "." || fileName == ".." {
		return ProcessingJob{}, ErrNotRawKey
	}
	base := strings.TrimSuffix(fileName, path.Ext(fileName))
	return ProcessingJob{
		Bucket:       bucket,
		RawKey:       rawKey,
		LinkID:       linkID,
		FileName:     fileName,
		ProcessedKey: linkID + "/processed/" + base + ".zip",
	}, nil
}
