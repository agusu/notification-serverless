package dynamodb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"serverless-notification/domain/notification"
)

var (
	ErrCreatingNotification = errors.New("failed to create notification")
	ErrStoringNotification  = errors.New("failed to store notification in dynamodb")
	ErrPagination           = errors.New("failed to paginate notifications")
	ErrNotificationNotFound = errors.New("notification not found")
)

type NotificationRepository struct {
	client    *dynamodb.Client
	tableName string
}

type NotificationItem struct {
	PK          string `dynamodbav:"PK"`     // USER#<userID>
	SK          string `dynamodbav:"SK"`     // NOTIF#<ISO8601_timestamp>#<ulid>
	GSI1PK      string `dynamodbav:"GSI1PK"` // NOTIF#<id>
	GSI1SK      string `dynamodbav:"GSI1SK"` // <ISO8601_timestamp>#<ulid>
	ID          string `dynamodbav:"id"`
	UserID      string `dynamodbav:"user_id"`
	Title       string `dynamodbav:"title"`
	Content     string `dynamodbav:"content"`
	ChannelName string `dynamodbav:"channel_name"`
	CreatedAt   string `dynamodbav:"created_at"` // ISO8601 string
	UpdatedAt   string `dynamodbav:"updated_at"` // ISO8601 string
	DeletedAt   string `dynamodbav:"deleted_at"` // ISO8601 string
}

// Constructor
func NewNotificationRepository(client *dynamodb.Client, tableName string) *NotificationRepository {
	return &NotificationRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	item := toItem(n)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreatingNotification, err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrStoringNotification, err)
	}

	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*notification.Notification, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "NOTIF#" + id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, ErrNotificationNotFound
	}

	var item NotificationItem
	err = attributevalue.UnmarshalMap(result.Items[0], &item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification: %w", err)
	}

	if item.DeletedAt != "" {
		return nil, ErrNotificationNotFound
	}

	return toEntity(item)
}

func (r *NotificationRepository) List(ctx context.Context, query notification.ListQuery) (*notification.ListResponse, error) {
	lastKey, err := decodeLastKey(query.NextToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPagination, err)
	}
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "USER#" + query.UserID},
		},
		Limit:             aws.Int32(int32(query.Limit)),
		ExclusiveStartKey: lastKey,
		FilterExpression:  aws.String("attribute_not_exists(deleted_at)"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	if len(result.Items) == 0 {
		return &notification.ListResponse{
			Notifications: []*notification.Notification{},
			NextToken:     "",
			HasMore:       false,
		}, nil
	}

	var items []NotificationItem
	err = attributevalue.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal notifications: %w", err)
	}

	nextToken, err := encodeLastKey(result.LastEvaluatedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode pagination token: %w", err)
	}
	entities, err := toEntities(items)
	if err != nil {
		return nil, fmt.Errorf("failed to convert items to entities: %w", err)
	}
	return &notification.ListResponse{
		Notifications: entities,
		NextToken:     nextToken,
		HasMore:       result.LastEvaluatedKey != nil,
	}, nil
}

func (r *NotificationRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	var setParts []string
	expressionValues := make(map[string]types.AttributeValue)

	blacklist := []string{"id", "user_id", "created_at", "channel_name"}

	for field, value := range updates {
		if slices.Contains(blacklist, field) {
			return fmt.Errorf("field %s is immutable and cannot be updated", field)
		}
		setParts = append(setParts, fmt.Sprintf("%s = :%s", field, field))
		av, err := attributevalue.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal update value: %w", err)
		}
		expressionValues[":"+field] = av
	}

	setParts = append(setParts, "updated_at = :updated_at")
	expressionValues[":updated_at"] = &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)}

	updateExpression := "SET " + strings.Join(setParts, ", ")

	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + existing.UserID},
			"SK": &types.AttributeValueMemberS{Value: "NOTIF#" + existing.CreatedAt.Format(time.RFC3339) + "#" + existing.ID},
		},
		UpdateExpression:          aws.String(updateExpression),
		ConditionExpression:       aws.String("attribute_exists(PK)"),
		ExpressionAttributeValues: expressionValues,
	})
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}
	return nil

}

func (r *NotificationRepository) Delete(ctx context.Context, id string) error {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	expressionValues := map[string]types.AttributeValue{
		":deleted_at": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + existing.UserID},
			"SK": &types.AttributeValueMemberS{Value: "NOTIF#" + existing.CreatedAt.Format(time.RFC3339) + "#" + existing.ID},
		},
		UpdateExpression:          aws.String("SET deleted_at = :deleted_at"),
		ExpressionAttributeValues: expressionValues,
		ConditionExpression:       aws.String("attribute_not_exists(deleted_at)"),
	})
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	return nil
}

func toItem(n *notification.Notification) NotificationItem {
	return NotificationItem{
		PK:          "USER#" + n.UserID,
		SK:          "NOTIF#" + n.CreatedAt.Format(time.RFC3339) + "#" + n.ID,
		GSI1PK:      "NOTIF#" + n.ID,
		GSI1SK:      "NOTIF#" + n.ID,
		ID:          n.ID,
		UserID:      n.UserID,
		Title:       n.Title,
		Content:     n.Content,
		ChannelName: n.ChannelName,
		CreatedAt:   n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   n.UpdatedAt.Format(time.RFC3339),
	}
}

func toEntities(items []NotificationItem) ([]*notification.Notification, error) {
	entities := make([]*notification.Notification, len(items))
	for i, item := range items {
		entity, err := toEntity(item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert notification: %w", err)
		}
		entities[i] = entity
	}
	return entities, nil
}

func toEntity(item NotificationItem) (*notification.Notification, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return &notification.Notification{
		ID:          item.ID,
		UserID:      item.UserID,
		Title:       item.Title,
		Content:     item.Content,
		ChannelName: item.ChannelName,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// Encode: DynamoDB map -> string
func encodeLastKey(lastKey map[string]types.AttributeValue) (string, error) {
	if len(lastKey) == 0 {
		return "", nil
	}

	pk, ok := lastKey["PK"].(*types.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("invalid PK type in last key")
	}
	sk, ok := lastKey["SK"].(*types.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("invalid SK type in last key")
	}

	simpleKey := map[string]string{
		"PK": pk.Value,
		"SK": sk.Value,
	}

	jsonBytes, err := json.Marshal(simpleKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pagination key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

// Decode: string -> DynamoDB map
func decodeLastKey(token string) (map[string]types.AttributeValue, error) {
	if token == "" {
		return nil, nil
	}

	jsonBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination token: %w", err)
	}

	var simpleKey map[string]string
	err = json.Unmarshal(jsonBytes, &simpleKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal last key: %w", err)
	}

	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: simpleKey["PK"]},
		"SK": &types.AttributeValueMemberS{Value: simpleKey["SK"]},
	}, nil
}
