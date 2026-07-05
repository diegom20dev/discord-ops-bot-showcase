package platform

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DynamoDBDocumentRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBDocumentRepository(client *dynamodb.Client, tableName string) *DynamoDBDocumentRepository {
	return &DynamoDBDocumentRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *DynamoDBDocumentRepository) SaveDocument(ctx context.Context, doc *domain.Document) error {
	av, err := attributevalue.MarshalMap(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

func (r *DynamoDBDocumentRepository) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if resp.Item == nil {
		return nil, nil
	}

	var doc domain.Document
	err = attributevalue.UnmarshalMap(resp.Item, &doc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	return &doc, nil
}

func (r *DynamoDBDocumentRepository) ListDocumentsByUser(ctx context.Context, userID string) ([]*domain.Document, error) {
	resp, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &r.tableName,
		IndexName:              stringPtr("user_id-created_at-index"),
		KeyConditionExpression: stringPtr("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	var documents []*domain.Document
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &documents)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents: %w", err)
	}

	return documents, nil
}

func (r *DynamoDBDocumentRepository) UpdateDocument(ctx context.Context, doc *domain.Document) error {
	av, err := attributevalue.MarshalMap(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}
