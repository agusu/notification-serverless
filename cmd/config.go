package cmd

import (
	"context"
	"os"
	"serverless-notification/adapters/dynamodb"
	"serverless-notification/clients"
	"serverless-notification/domain/notification"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsDynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// InitDependencies initializes all dependencies and returns a configured Service
func InitDependencies() *notification.Service {
	cfg := loadAWSConfig()

	dynamoClient := awsDynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	notificationRepo := dynamodb.NewNotificationRepository(dynamoClient, os.Getenv("NOTIFICATIONS_TABLE"))
	queue := clients.NewSQSClient(sqsClient, os.Getenv("SQS_QUEUE_URL"))

	validator := &noopValidator{}

	service := notification.NewService(notificationRepo, queue, validator)

	return service
}

func loadAWSConfig() aws.Config {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Default region
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}
	return cfg
}

// dummy validator that always returns nil
type noopValidator struct{}

func (v *noopValidator) Validate(channelName string, meta map[string]string) error {
	return nil
}
