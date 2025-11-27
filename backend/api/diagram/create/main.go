package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/db"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

// Request defines the expected input body
type Request struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	UserID    string `json:"userId"`
}

// DiagramData defines the inner Data object
type DiagramData struct {
	ID        string `json:"id" dynamodbav:"id"`
	Name      string `json:"name" dynamodbav:"name"`
	CreatedBy string `json:"createdBy" dynamodbav:"createdBy"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
}

// DiagramEntity defines the DynamoDB item structure
type DiagramEntity struct {
	PK   string      `dynamodbav:"PK"`
	SK   string      `dynamodbav:"SK"`
	Type string      `dynamodbav:"Type"`
	Data DiagramData `dynamodbav:"Data"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse Request Body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(400, generic.Json{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.ProjectID == "" || req.Name == "" || req.UserID == "" {
		return generic.Response(400, generic.Json{"error": "Missing required fields: projectId, name, userId"})
	}

	// 2. Create Diagram Entity
	diagramID := uuid.New().String()
	timestamp := time.Now().UTC().Format(time.RFC3339)

	entity := DiagramEntity{
		PK:   fmt.Sprintf("PROJECT#%s", req.ProjectID),
		SK:   fmt.Sprintf("DIAGRAM#%s", diagramID),
		Type: "DIAGRAM",
		Data: DiagramData{
			ID:        diagramID,
			Name:      req.Name,
			CreatedBy: req.UserID,
			CreatedAt: timestamp,
		},
	}

	// 3. Marshal to DynamoDB AttributeValues
	item, err := attributevalue.MarshalMap(entity)
	if err != nil {
		fmt.Printf("Error marshalling item: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Internal server error"})
	}

	// 4. Get DynamoDB Client
	svc, err := db.GetClient(ctx)
	if err != nil {
		fmt.Printf("Error getting DB client: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Internal server error"})
	}

	// 5. Put Item
	tableName := "application-data" 
	_, err = svc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		fmt.Printf("Error putting item to DynamoDB: %v\n", err)
		return generic.Response(500, generic.Json{"error": "Failed to save diagram"})
	}

	// 6. Return Success
	return generic.Response(201, generic.Json{
		"message": "Diagram created successfully",
		"diagram": entity.Data,
	})
}
