package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBRateLimiter struct {
	client    *dynamodb.Client
	tableName string
	limits    map[string]RateLimit
}

type RateLimit struct {
	MaxActions int
	WindowSecs int
}

type RateLimitRecord struct {
	UserID    string    `dynamodbav:"user_id"`
	Action    string    `dynamodbav:"action"`
	Count     int       `dynamodbav:"count"`
	ResetTime time.Time `dynamodbav:"reset_time"`
}

func NewDynamoDBRateLimiter(client *dynamodb.Client, tableName string) *DynamoDBRateLimiter {
	return &DynamoDBRateLimiter{
		client:    client,
		tableName: tableName,
		limits: map[string]RateLimit{
			"summarize": {MaxActions: 10, WindowSecs: 3600}, // 10 per hour
			"capture":   {MaxActions: 50, WindowSecs: 3600}, // 50 per hour
		},
	}
}

func (rl *DynamoDBRateLimiter) AllowAction(ctx context.Context, userID, action string) (bool, error) {
	limit, exists := rl.limits[action]
	if !exists {
		return true, nil
	}

	key := fmt.Sprintf("%s#%s", userID, action)

	getResp, err := rl.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &rl.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to get rate limit record: %w", err)
	}

	now := time.Now()
	record := &RateLimitRecord{
		UserID:    userID,
		Action:    action,
		Count:     1,
		ResetTime: now.Add(time.Duration(limit.WindowSecs) * time.Second),
	}

	if getResp.Item != nil {
		var existing RateLimitRecord
		if err := attributevalue.UnmarshalMap(getResp.Item, &existing); err != nil {
			return false, fmt.Errorf("failed to unmarshal record: %w", err)
		}

		if now.Before(existing.ResetTime) {
			if existing.Count >= limit.MaxActions {
				return false, fmt.Errorf("rate limit exceeded")
			}
			record.Count = existing.Count + 1
			record.ResetTime = existing.ResetTime
		}
	}

	av, err := attributevalue.MarshalMap(record)
	if err != nil {
		return false, fmt.Errorf("failed to marshal record: %w", err)
	}
	av["id"] = &types.AttributeValueMemberS{Value: key}

	_, err = rl.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &rl.tableName,
		Item:      av,
	})
	if err != nil {
		return false, fmt.Errorf("failed to update rate limit: %w", err)
	}

	return record.Count <= limit.MaxActions, nil
}
