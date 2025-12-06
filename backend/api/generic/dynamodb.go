package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

// Put item into DynamoDB PK, SK, Type, and Data map
// Source: ChatGPT
func PutItemInDynamoDB(
	ctx context.Context,
	client *dynamodb.Client,
	tableName string,
	pk string,
	sk string,
	itemType string,
	dataMap map[string]types.AttributeValue,
) error {

	item := map[string]types.AttributeValue{
		"PK":   &types.AttributeValueMemberS{Value: pk},
		"SK":   &types.AttributeValueMemberS{Value: sk},
		"Type": &types.AttributeValueMemberS{Value: itemType},
		"Data": &types.AttributeValueMemberM{Value: dataMap},
	}

	_, err := client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	return err
}

// Function to convert struct to DynamoDB attribute map
// Source: ChatGPT
func ConvertStructToAttributeValueMap(data any) (map[string]types.AttributeValue, error) {
	marshaled, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = json.Unmarshal(marshaled, &m)
	if err != nil {
		return nil, err
	}
	result := make(map[string]types.AttributeValue)
	for k, v := range m {
		result[k] = convertToAttributeValue(v)
	}
	return result, nil
}

// Internal function to recursively convert any to DynamoDB AttributeValue
// Source: ChatGPT
func convertToAttributeValue(v any) types.AttributeValue {
	switch val := v.(type) {
	case string:
		return &types.AttributeValueMemberS{Value: val}
	case float64:
		return &types.AttributeValueMemberN{Value: fmt.Sprintf("%v", val)}
	case bool:
		return &types.AttributeValueMemberBOOL{Value: val}
	case []any:
		avList := make([]types.AttributeValue, len(val))
		for i, e := range val {
			avList[i] = convertToAttributeValue(e)
		}
		return &types.AttributeValueMemberL{Value: avList}
	case []string:
		avList := make([]types.AttributeValue, len(val))
		for i, e := range val {
			avList[i] = &types.AttributeValueMemberS{Value: e}
		}
		return &types.AttributeValueMemberL{Value: avList}
	case map[string]any:
		avMap := make(map[string]types.AttributeValue)
		for k, e := range val {
			avMap[k] = convertToAttributeValue(e)
		}
		return &types.AttributeValueMemberM{Value: avMap}
	default:
		return &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", val)}
	}
}
