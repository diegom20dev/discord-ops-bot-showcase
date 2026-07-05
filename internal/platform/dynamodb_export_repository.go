package platform

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DynamoDBExportRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBExportRepository(client *dynamodb.Client, tableName string) *DynamoDBExportRepository {
	return &DynamoDBExportRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *DynamoDBExportRepository) SaveExport(ctx context.Context, export *domain.Export) error {
	av, err := attributevalue.MarshalMap(export)
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to save export: %w", err)
	}

	return nil
}

func (r *DynamoDBExportRepository) GetExport(ctx context.Context, id string) (*domain.Export, error) {
	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get export: %w", err)
	}

	if resp.Item == nil {
		return nil, nil
	}

	var export domain.Export
	err = attributevalue.UnmarshalMap(resp.Item, &export)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal export: %w", err)
	}

	return &export, nil
}

func (r *DynamoDBExportRepository) UpdateExport(ctx context.Context, export *domain.Export) error {
	av, err := attributevalue.MarshalMap(export)
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to update export: %w", err)
	}

	return nil
}
