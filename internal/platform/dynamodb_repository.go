package platform

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBRepository(client *dynamodb.Client, tableName string) *DynamoDBRepository {
	return &DynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *DynamoDBRepository) SaveCapture(ctx context.Context, capture *domain.Capture) error {
	av, err := attributevalue.MarshalMap(capture)
	if err != nil {
		return fmt.Errorf("failed to marshal capture: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

func (r *DynamoDBRepository) GetCapture(ctx context.Context, id string) (*domain.Capture, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("capture not found")
	}

	var capture domain.Capture
	err = attributevalue.UnmarshalMap(result.Item, &capture)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal capture: %w", err)
	}

	return &capture, nil
}

func (r *DynamoDBRepository) ListCapturesByUser(ctx context.Context, userID string) ([]*domain.Capture, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("user_id-created_at-index"),
		KeyConditionExpression: aws.String("user_id = :user_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":user_id": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	var captures []*domain.Capture
	err = attributevalue.UnmarshalListOfMaps(result.Items, &captures)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal captures: %w", err)
	}

	return captures, nil
}

func (r *DynamoDBRepository) UpdateCapture(ctx context.Context, capture *domain.Capture) error {
	// Use PUT instead of UPDATE for simplicity (replaces entire item)
	av, err := attributevalue.MarshalMap(capture)
	if err != nil {
		return fmt.Errorf("failed to marshal capture: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}
