package main

import (
	"context"
	"fmt"

	"github.com/Alberta2514640/clutter/backend/api/db"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" {
		return generic.Response(400, generic.Json{"error": "Missing required query parameter: projectId"})
	}

	svc, err := db.GetClient(ctx)
	if err != nil {
		fmt.Printf("Error getting DB client: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Internal server error"})
	}

	tableName := "application-data"

	// Case 1: Get Single Diagram
	if diagramID != "" {
		out, err := svc.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
				"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
			},
		})
		if err != nil {
			fmt.Printf("Error getting item: %v\n", err)
			return generic.Response(500, generic.Json{"error": "Failed to fetch diagram"})
		}

		if out.Item == nil {
			return generic.Response(404, generic.Json{"error": "Diagram not found"})
		}

		var entity map[string]interface{}
		if err := attributevalue.UnmarshalMap(out.Item, &entity); err != nil {
			fmt.Printf("Error unmarshalling item: %v\n", err)
			return generic.Response(500, generic.Json{"error": "Internal server error"})
		}

		return generic.Response(200, generic.Json{"diagram": entity["Data"]})
	}

	// Case 2: List Diagrams for Project
	// Query PK = PROJECT#<projectId> and SK begins_with DIAGRAM#
	out, err := svc.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			":sk": &types.AttributeValueMemberS{Value: "DIAGRAM#"},
		},
	})
	if err != nil {
		fmt.Printf("Error querying items: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Failed to list diagrams"})
	}

	var diagrams []interface{}
	for _, item := range out.Items {
		var entity map[string]interface{}
		if err := attributevalue.UnmarshalMap(item, &entity); err == nil {
			if data, ok := entity["Data"]; ok {
				diagrams = append(diagrams, data)
			}
		}
	}

	return generic.Response(200, generic.Json{"diagrams": diagrams})
}
