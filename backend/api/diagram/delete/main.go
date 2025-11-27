package main

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/db"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" || diagramID == "" {
		return generic.Response(400, generic.Json{"error": "Missing required query parameters: projectId, diagramId"})
	}

	svc, err := db.GetClient(ctx)
	if err != nil {
		fmt.Printf("Error getting DB client: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Internal server error"})
	}

	tableName := "application-data"

	_, err = svc.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
	})

	if err != nil {
		fmt.Printf("Error deleting item: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Failed to delete diagram"})
	}

	return generic.Response(200, generic.Json{"message": "Diagram deleted successfully"})
}
