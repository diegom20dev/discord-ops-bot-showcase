package platform

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DynamoDBReminderRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBReminderRepository(client *dynamodb.Client, tableName string) *DynamoDBReminderRepository {
	return &DynamoDBReminderRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *DynamoDBReminderRepository) SaveReminder(ctx context.Context, reminder *domain.Reminder) error {
	av, err := attributevalue.MarshalMap(reminder)
	if err != nil {
		return fmt.Errorf("failed to marshal reminder: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to save reminder: %w", err)
	}

	return nil
}

func (r *DynamoDBReminderRepository) GetReminder(ctx context.Context, id string) (*domain.Reminder, error) {
	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	if resp.Item == nil {
		return nil, nil
	}

	var reminder domain.Reminder
	err = attributevalue.UnmarshalMap(resp.Item, &reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminder: %w", err)
	}

	return &reminder, nil
}

func (r *DynamoDBReminderRepository) ListRemindersByUser(ctx context.Context, userID string) ([]*domain.Reminder, error) {
	resp, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              stringPtr("user_id-scheduled_at-index"),
		KeyConditionExpression: stringPtr("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}

	var reminders []*domain.Reminder
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &reminders)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}

	return reminders, nil
}

func (r *DynamoDBReminderRepository) ListPendingReminders(ctx context.Context) ([]*domain.Reminder, error) {
	resp, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        &r.tableName,
		FilterExpression: stringPtr("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "pending"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan reminders: %w", err)
	}

	var reminders []*domain.Reminder
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &reminders)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}

	return reminders, nil
}

func (r *DynamoDBReminderRepository) UpdateReminder(ctx context.Context, reminder *domain.Reminder) error {
	av, err := attributevalue.MarshalMap(reminder)
	if err != nil {
		return fmt.Errorf("failed to marshal reminder: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to update reminder: %w", err)
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}
