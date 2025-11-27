package db

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var (
	client *dynamodb.Client
	once   sync.Once
)

// GetClient returns a singleton DynamoDB client
func GetClient(ctx context.Context) (*dynamodb.Client, error) {
	var err error
	once.Do(func() {
		cfg, loadErr := config.LoadDefaultConfig(ctx)
		if loadErr != nil {
			err = loadErr
			return
		}
		client = dynamodb.NewFromConfig(cfg)
	})
	return client, err
}
