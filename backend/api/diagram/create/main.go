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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

// Request defines the expected input body
type Request struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
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
	// 1. Extract userId from request
	userID, err := getUserIDFromRequest(request)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "UNAUTHORIZED",
				"message": err.Error(),
			},
		})
	}

	// 2. Parse Request Body
	var req Request
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "Invalid request body",
			},
		})
	}

	// 3. Validate Required Fields
	var missingFields []string
	if req.ProjectID == "" {
		missingFields = append(missingFields, "projectId")
	}
	if req.Name == "" {
		missingFields = append(missingFields, "name")
	}
	if len(missingFields) > 0 {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": fmt.Sprintf("Missing required fields: %v", missingFields),
			},
		})
	}

	// 4. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganization(ctx, ddb, tableName, req.ProjectID)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    authErr.Code,
					"message": authErr.Message,
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch project",
			},
		})
	}

	// 5. Check organization membership
	if err := generic.CheckOrganizationMembership(ctx, ddb, tableName, userID, orgID); err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			return generic.Response(authErr.StatusCode, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    authErr.Code,
					"message": authErr.Message,
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check authorization",
			},
		})
	}

	// 6. Create Diagram Entity
	diagramID := uuid.NewString()
	timestamp := time.Now().UTC().Format(time.RFC3339)

	entity := DiagramEntity{
		PK:   fmt.Sprintf("PROJECT#%s", req.ProjectID),
		SK:   fmt.Sprintf("DIAGRAM#%s", diagramID),
		Type: "DIAGRAM",
		Data: DiagramData{
			ID:        diagramID,
			Name:      req.Name,
			CreatedBy: userID,
			CreatedAt: timestamp,
		},
	}

	// 7. Marshal to DynamoDB AttributeValues
	item, err := attributevalue.MarshalMap(entity)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Internal server error",
			},
		})
	}

	// 8. Put Item
	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to save diagram",
			},
		})
	}

	// 9. Return Success
	return generic.Response(http.StatusCreated, generic.Json{
		"success": true,
		"data":    entity.Data,
	})
}

// getUserIDFromRequest pulls user identity from authorizer or headers
func getUserIDFromRequest(req events.APIGatewayProxyRequest) (string, error) {
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

	for k, v := range req.Headers {
		if strings.ToLower(k) == "x-user-id" && v != "" {
			return v, nil
		}
	}

	return "", errors.New("missing user identity in request context")
}
