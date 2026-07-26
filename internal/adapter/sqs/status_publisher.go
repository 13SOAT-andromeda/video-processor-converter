package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/13SOAT-andromeda/video-processor-converter/internal/application/ports"
	"github.com/13SOAT-andromeda/video-processor-converter/internal/domain"
)

var _ ports.StatusPublisher = (*StatusPublisher)(nil)

type StatusPublisher struct {
	client   *awssqs.Client
	queueURL string
}

func NewStatusPublisher(client *awssqs.Client, queueURL string) *StatusPublisher {
	return &StatusPublisher{client: client, queueURL: queueURL}
}

func (p *StatusPublisher) Publish(ctx context.Context, e domain.StatusEvent) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal status event: %w", err)
	}
	if _, err := p.client.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(body)),
	}); err != nil {
		return fmt.Errorf("sqs send %s: %w", e.Status, err)
	}
	return nil
}
