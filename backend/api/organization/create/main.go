package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
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

// Incoming request body structure
type requestBody struct {
	OrganizationName string `json:"organizationName"`
	Description      string `json:"description"`
}

// DynamoDB item structure
type organizationData struct {
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	CreatedBy     string   `json:"createdBy"`
	CreatedAt     string   `json:"createdAt"`
	Description   string   `json:"description"`
	Members       []string `json:"members"`
	TotalProjects int      `json:"totalProjects"`
	TotalDiagrams int      `json:"totalDiagrams"`
}

func handler(request events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}

	// Process body from API Gateway request
	var body requestBody

	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	organizationName := body.OrganizationName
	description := body.Description

	// Generate required fields for organization
	organizationId := uuid.NewString()
	timeNow := time.Now().UTC().Format(time.RFC3339)
	members := []string{userData.Email}

	// Populate organization data struct
	organizationData := organizationData{
		Id:            organizationId,
		Name:          organizationName,
		CreatedBy:     userData.Email,
		CreatedAt:     timeNow,
		Description:   description,
		Members:       members,
		TotalProjects: 0,
		TotalDiagrams: 0,
	}

	// Convert organizationData struct to AttributeValue map for DynamoDB
	organizationAttributeValueMap, err := generic.ConvertStructToAttributeValueMap(organizationData)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to convert 'organizationData' struct to an Attribute Value map", "error": err.Error()},
		)
	}

	// Put organization item into DynamoDB
	ctx := context.Background()
	pk := fmt.Sprintf("USER#%s", userData.Email)
	sk := fmt.Sprintf("ORG#%s", organizationId)
	itemType := "ORGANIZATION"
	err = generic.PutItemInDynamoDB(
		ctx,
		ddb,
		tableName,
		pk,
		sk,
		itemType,
		organizationAttributeValueMap,
	)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": fmt.Sprintf("failed to put organization into %s DynamoDB table", tableName), "error": err.Error()},
		)
	}

	return generic.Response(
		http.StatusOK,
		generic.Json{"message": "new organization created successfully", "organizationData": organizationData},
	)

}
