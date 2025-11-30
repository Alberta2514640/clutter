package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ddb       *dynamodb.Client
	tableName string
)

func init() {
	var err error
	ddb, tableName, err = generic.GetDynamodbClient()
	if err != nil {
		panic("failed to initialize DynamoDB client: " + err.Error())
	}
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" || diagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "Missing required query parameters: projectId, diagramId"})
	}

	_, err := ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})

	if err != nil {
		fmt.Printf("Error deleting item: %v\n", err)
		// Check if the error is due to condition not being met (item doesn't exist)
		var condErr *types.ConditionalCheckFailedException
		if ok := errors.As(err, &condErr); ok {
			return generic.Response(http.StatusNotFound, generic.Json{"error": "Diagram not found"})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "Failed to delete diagram"})
	}

	return generic.Response(http.StatusOK, generic.Json{"message": "Diagram deleted successfully"})
}
