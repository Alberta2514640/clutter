package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UpdateRequest struct {
	ProjectID string                 `json:"projectId"`
	DiagramID string                 `json:"diagramId"`
	Name      *string                `json:"name,omitempty"`
	UILayout  map[string]interface{} `json:"uiLayout,omitempty"`
}

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
	var req UpdateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "Invalid request body"})
	}

	if req.ProjectID == "" || req.DiagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "Missing required fields: projectId, diagramId"})
	}

	pk := fmt.Sprintf("PROJECT#%s", req.ProjectID)
	sk := fmt.Sprintf("DIAGRAM#%s", req.DiagramID)

	// Build Update Expression
	updateExpr := "SET #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#updatedAt": "Data.updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}

	if req.Name != nil {
		updateExpr += ", #name = :name"
		exprAttrNames["#name"] = "Data.name"
		exprAttrValues[":name"] = &types.AttributeValueMemberS{Value: *req.Name}
	}

	// If we had a UI layout field in Data, we would update it here.
	// Assuming Data.uiLayout exists or we are adding it.
	if req.UILayout != nil {
		updateExpr += ", #uiLayout = :uiLayout"
		exprAttrNames["#uiLayout"] = "Data.uiLayout"

		// Marshal the map to AttributeValue
		av, err := attributevalue.Marshal(req.UILayout)
		if err != nil {
			fmt.Printf("Error marshalling uiLayout: %v\n", err)
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": "Internal server error"})
		}
		exprAttrValues[":uiLayout"] = av
	}

	_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK)"),
		ReturnValues:              types.ReturnValueAllNew,
	})

	if err != nil {
		fmt.Printf("Error updating item: %v\n", err)
		// Check if the error is due to condition not being met (item doesn't exist)
		var condErr *types.ConditionalCheckFailedException
		if ok := errors.As(err, &condErr); ok {
			return generic.Response(http.StatusNotFound, generic.Json{"error": "Diagram not found"})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "Failed to update diagram"})
	}

	return generic.Response(http.StatusOK, generic.Json{"message": "Diagram updated successfully"})
}
