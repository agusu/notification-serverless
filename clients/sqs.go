package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"serverless-notification/domain/notification"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSClient implements the notification.Queue to send messages to Amazon SQS
type SQSClient struct {
	client   *sqs.Client
	queueURL string
}

// NewSQSClient creates a new SQSClient
func NewSQSClient(client *sqs.Client, queueURL string) *SQSClient {
	return &SQSClient{
		client:   client,
		queueURL: queueURL,
	}
}

// Publish sends a notification to the SQS queue for asynchronous processing
func (c *SQSClient) Publish(ctx context.Context, msg *notification.DispatchMessage) error {
	if c.queueURL == "" {
		return fmt.Errorf("queue URL is not set")
	}

	messageJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:               aws.String(c.queueURL),
		MessageBody:            aws.String(string(messageJSON)),
		MessageGroupId:         aws.String(msg.UserID),
		MessageDeduplicationId: aws.String(msg.NotificationID),
	}

	output, err := c.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}
	log.Printf("Message sent to SQS: %s", *output.MessageId)

	return nil
}
