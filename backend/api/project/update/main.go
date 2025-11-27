package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UpdateProjectRequest struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

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
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}

	ddb = dynamodb.NewFromConfig(cfg)

	tableName = os.Getenv("DDB_TABLE_NAME")
	if tableName == "" {
		tableName = "application-data"
	}
}

func main() {
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()

	// 1. Require user identity (authorizer or x-user-id header)
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

	// 2. Query param: projectId (required)
	projectID := req.QueryStringParameters["projectId"]
	if projectID == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "projectId query parameter is required",
			},
		})
	}

	// 3. Parse and validate JSON body
	var body UpdateProjectRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INVALID_JSON",
				"message": "invalid JSON body",
			},
		})
	}

	if body.OrganizationID == "" || body.Name == "" {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "VALIDATION_ERROR",
				"message": "organizationId and name are required",
			},
		})
	}

	// 4. Build key
	orgID := body.OrganizationID
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{
			Value: fmt.Sprintf("ORG#%s", orgID),
		},
		"SK": &types.AttributeValueMemberS{
			Value: fmt.Sprintf("PROJECT#%s", projectID),
		},
	}

	// 5. Update the item (name, description, updatedAt)
	now := time.Now().UTC().Format(time.RFC3339)

	updateExpr := "SET #d.#name = :name, #d.#description = :description, #d.#updatedAt = :updatedAt"
	exprNames := map[string]string{
		"#d":          "Data",
		"#name":       "name",
		"#description": "description",
		"#updatedAt":  "updatedAt",
	}
	exprValues := map[string]types.AttributeValue{
		":name":       &types.AttributeValueMemberS{Value: body.Name},
		":description": &types.AttributeValueMemberS{Value: body.Description},
		":updatedAt":  &types.AttributeValueMemberS{Value: now},
	}

	out, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       key,
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			// Item does not exist
			return generic.Response(http.StatusNotFound, generic.Json{
				"success": false,
				"error": generic.Json{
					"code":    "NOT_FOUND",
					"message": "project not found",
				},
			})
		}

		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "failed to update project",
			},
		})
	}

	// 6. Decode updated item into Project struct
	dataAttr, ok := out.Attributes["Data"].(*types.AttributeValueMemberM)
	if !ok {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"success": false,
			"error": generic.Json{
				"code":    "INTERNAL_ERROR",
				"message": "invalid project item format",
			},
		})
	}
	data := dataAttr.Value

	project := Project{
		ID:             getStringAttr(data, "id"),
		OrganizationID: orgID,
		Name:           getStringAttr(data, "name"),
		Description:    getStringAttr(data, "description"),
		CreatedBy:      getStringAttr(data, "createdBy"),
		CreatedAt:      getStringAttr(data, "createdAt"),
		UpdatedAt:      getStringAttr(data, "updatedAt"),
	}

	// We don't use userID here yet, but it's validated; could be used for audit later
	_ = userID

	return generic.Response(http.StatusOK, generic.Json{
		"success": true,
		"data":    project,
	})
}

func getStringAttr(m map[string]types.AttributeValue, key string) string {
	if v, ok := m[key]; ok {
		if s, ok2 := v.(*types.AttributeValueMemberS); ok2 {
			return s.Value
		}
	}
	return ""
}

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