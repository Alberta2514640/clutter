package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

	projectID := request.QueryStringParameters["projectId"]
	diagramID := request.QueryStringParameters["diagramId"]

	if projectID == "" || diagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required query parameters: projectId, diagramId",
			},
		})
	}

	// 2. Authorization: Get project's organization and check membership
	orgID, err := generic.GetProjectOrganization(ctx, ddb, tableName, projectID)
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

	// 3. Check organization membership
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

	// 4. Delete diagram
	_, err = ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})

	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if ok := errors.As(err, &condErr); ok {
			return generic.Response(http.StatusNotFound, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "NOT_FOUND",
					"message": "Diagram not found",
				},
			})
		}
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete diagram",
			},
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"message": "Diagram deleted successfully",
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
