package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// Incoming request body
type CreateProjectRequest struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

// Project shape we send back to frontend
type Project struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CreatedBy      string `json:"createdBy"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

var (
	ddb       *dynamodb.Client
	tableName string
)

func init() {
    var err error
    ddb, tableName, err = generic.GetDynamodbClient()
    if err != nil {
        panic(fmt.Sprintf("failed to initialize DynamoDB client: %v", err))
    }
}

func main() {
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()

	// 1. Parse JSON body
	var body CreateProjectRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "invalid JSON body",
			},
		})
	}

	// 3. Get userId from authorizer (or header for local testing)
	userID, err := getUserIDFromRequest(req)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "UNAUTHORIZED",
				"message": err.Error(),
			},
		})
	}

	// 4. Generate projectId and timestamps
	projectID := "proj-" + uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	project := Project{
		ID:             projectID,
		OrganizationID: body.OrganizationID,
		Name:           body.Name,
		Description:    body.Description,
		CreatedBy:      userID,
		CreatedAt:      now,
	}

	// 5. Build DynamoDB item
	item := map[string]types.AttributeValue{
		"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", body.OrganizationID)},
		"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
		"Type": &types.AttributeValueMemberS{Value: "PROJECT"},
		"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: project.ID},
			"name":        &types.AttributeValueMemberS{Value: project.Name},
			"description": &types.AttributeValueMemberS{Value: project.Description},
			"createdBy":   &types.AttributeValueMemberS{Value: project.CreatedBy},
			"createdAt":   &types.AttributeValueMemberS{Value: project.CreatedAt},
		}},
	}

	// 6. PutItem into DynamoDB
	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "failed to create project",
			},
		})
	}

	// 7. Return created project
	return generic.Response(http.StatusCreated, generic.Json{
		"success": true,
		"data":    project,
	})
}

// getUserIDFromRequest pulls user identity from authorizer or headers
func getUserIDFromRequest(req events.APIGatewayProxyRequest) (string, error) {
	// 1. Check custom authorizer / Cognito (future-ready)
	if req.RequestContext.Authorizer != nil {
		if v, ok := req.RequestContext.Authorizer["userId"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
		if v, ok := req.RequestContext.Authorizer["sub"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
		if v, ok := req.RequestContext.Authorizer["email"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				return s, nil
			}
		}
	}

	// 2. Local / simple testing: case-insensitive x-user-id header
	for k, v := range req.Headers {
		if strings.ToLower(k) == "x-user-id" && v != "" {
			return v, nil
		}
	}

	return "", errors.New("missing user identity in request context")
}