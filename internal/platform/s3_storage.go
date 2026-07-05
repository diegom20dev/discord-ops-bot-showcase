package platform

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type S3Storage struct {
	client     *s3.Client
	bucketName string
}

func NewS3Storage(client *s3.Client, bucketName, _ string) *S3Storage {
	return &S3Storage{
		client:     client,
		bucketName: bucketName,
	}
}

func (s *S3Storage) Upload(ctx context.Context, fileName string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(data),
		ContentType: aws.String(getContentType(fileName)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate public URL (S3 can infer region from bucket)
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucketName, fileName)
	return url, nil
}

func getContentType(fileName string) string {
	if len(fileName) > 4 {
		ext := fileName[len(fileName)-4:]
		switch ext {
		case ".csv":
			return "text/csv; charset=utf-8"
		case ".md":
			return "text/plain; charset=utf-8"
		case ".txt":
			return "text/plain; charset=utf-8"
		case ".pdf":
			return "application/pdf"
		}
	}
	if len(fileName) > 5 {
		ext := fileName[len(fileName)-5:]
		if ext == ".json" {
			return "application/json; charset=utf-8"
		}
	}
	return "text/plain; charset=utf-8"
}
