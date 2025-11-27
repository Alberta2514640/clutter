package generic

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func GetDynamodbClient() (*dynamodb.Client, string, error) {

	// Get AWS credentials through the Lambda container
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, "", err
	}

	// Get tableName from environment variables
	tableName := os.Getenv("DDB_TABLE_NAME")
	if tableName == "" {
		return nil, "", fmt.Errorf("DDB_TABLE_NAME environment variable is not set")
	}

	// Create a DynamoDB client using the config loaded from Lambda container
	ddb := dynamodb.NewFromConfig(cfg)
	return ddb, tableName, nil

}
