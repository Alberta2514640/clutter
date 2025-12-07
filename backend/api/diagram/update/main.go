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

	// 2. Parse request body
	var req UpdateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "Invalid request body",
			},
		})
	}

	if req.ProjectID == "" || req.DiagramID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "Missing required fields: projectId, diagramId",
			},
		})
	}

	// 3. Authorization: Get project's organization and check membership
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

	// 4. Check organization membership
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

	// 5. Build Update Expression
	// NOTE: "Data" is a DynamoDB reserved keyword, so we must use expression attribute names (#Data)
	// to reference nested attributes within the Data map. The correct syntax is #Data.#fieldName.
	pk := fmt.Sprintf("PROJECT#%s", req.ProjectID)
	sk := fmt.Sprintf("DIAGRAM#%s", req.DiagramID)

	updateExpr := "SET #Data.#updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#Data":      "Data",
		"#updatedAt": "updatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}

	if req.Name != nil {
		updateExpr += ", #Data.#name = :name"
		exprAttrNames["#name"] = "name"
		exprAttrValues[":name"] = &types.AttributeValueMemberS{Value: *req.Name}
	}

	if req.UILayout != nil {
		updateExpr += ", #Data.#uiLayout = :uiLayout"
		exprAttrNames["#uiLayout"] = "uiLayout"

		av, err := attributevalue.Marshal(req.UILayout)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "INTERNAL_ERROR",
					"message": "Internal server error",
				},
			})
		}
		exprAttrValues[":uiLayout"] = av
	}

	// 6. Update item
	_, err = ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
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
				"message": "Failed to update diagram",
			},
		})
	}

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"message": "Diagram updated successfully",
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
