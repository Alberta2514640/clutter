package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/log-in/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

// Incoming request body
type GoogleLoginRequest struct {
	Token string `json:"token"`
}

// Outgoing user structure
type UserData struct {
	Uuid             string `json:"uuid"`
	Email            string `json:"email"`
	Name             string `json:"name"`
	PictureUrl       string `json:"pictureUrl"`
	AccountCreatedOn string `json:"accountCreatedOn"`
}

var (
	ddb       *dynamodb.Client
	tableName string
)

// init() runs only once when the Lambda container is first created
// All objects created are reused accross invocations until container dies
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	ctx := context.Background()

	// Parse JSON body
	var req GoogleLoginRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"error": "invalid request body", "message": err.Error()})
	}

	// Validate Google ID token
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	payload, err := idtoken.Validate(ctx, req.Token, clientID)
	if err != nil {
		return generic.Response(http.StatusUnauthorized, generic.Json{"error": "invalid Google token", "message": err.Error()})
	}

	// Extract user info from claims
	emailFromGoogle := payload.Claims["email"].(string)
	nameFromGoogle := payload.Claims["name"].(string)
	pictureUrlFromGoogle := payload.Claims["picture"].(string)

	// Generate app JWT
	token, err := internal.GenerateJWT(emailFromGoogle, nameFromGoogle, pictureUrlFromGoogle)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to generate JWT token", "message": err.Error()})
	}

	// Make input object for DynamoDB request
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", emailFromGoogle)},
			"SK": &types.AttributeValueMemberS{Value: "PROFILE"},
		},
	}

	// Check if user is already created in DynamoDB table
	existingUserData, err := getExistingUser(ctx, getItemInput)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to query DynamoDB", "message": err.Error()})
	}

	// Check if user details have changed on existingUserData if it exists
	var emptyUserData UserData
	if existingUserData != emptyUserData {
		updatedExistingData, err := checkAndUpdateUserDetailsChanged(
			ctx,
			getItemInput,
			existingUserData,
			nameFromGoogle,
			pictureUrlFromGoogle,
		)
		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to update user details", "message": err.Error()})
		}

		return generic.Response(http.StatusCreated, generic.Json{
			"message":  "New user created successsfully",
			"token":    token,
			"userData": updatedExistingData,
		})

	}

	// Create required fields for new user
	uuid := uuid.NewString()
	timeNow := time.Now().UTC().Format(time.RFC3339)

	// Build newUserData struct
	newUserData := UserData{
		Uuid:             uuid,
		Email:            emailFromGoogle,
		Name:             nameFromGoogle,
		PictureUrl:       pictureUrlFromGoogle,
		AccountCreatedOn: timeNow,
	}

	// Build DynamoDB item
	item := map[string]types.AttributeValue{
		"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", emailFromGoogle)},
		"SK":   &types.AttributeValueMemberS{Value: "PROFILE"},
		"Type": &types.AttributeValueMemberS{Value: "USER"},
		"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"uuid":             &types.AttributeValueMemberS{Value: newUserData.Uuid},
			"email":            &types.AttributeValueMemberS{Value: newUserData.Email},
			"name":             &types.AttributeValueMemberS{Value: newUserData.Name},
			"pictureUrl":       &types.AttributeValueMemberS{Value: newUserData.PictureUrl},
			"accountCreatedOn": &types.AttributeValueMemberS{Value: newUserData.AccountCreatedOn},
		}},
	}

	// Insert new user into DynamoDB table
	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{"error": "failed to insert user", "message": err.Error()})
	}

	return generic.Response(http.StatusCreated, generic.Json{
		"message":  "New user created successfully",
		"token":    token,
		"userData": newUserData,
	})

}

func getExistingUser(ctx context.Context, getItemInput *dynamodb.GetItemInput) (UserData, error) {

	// Query the DynamoDB for the user
	result, err := ddb.GetItem(ctx, getItemInput)
	if err != nil {
		return UserData{}, err
	}
	// If the user exists, return the existing user data
	if result.Item != nil {

		existingData := result.Item["Data"].(*types.AttributeValueMemberM).Value

		existingUser := UserData{
			Uuid:             existingData["uuid"].(*types.AttributeValueMemberS).Value,
			Email:            existingData["email"].(*types.AttributeValueMemberS).Value,
			Name:             existingData["name"].(*types.AttributeValueMemberS).Value,
			PictureUrl:       existingData["pictureUrl"].(*types.AttributeValueMemberS).Value,
			AccountCreatedOn: existingData["accountCreatedOn"].(*types.AttributeValueMemberS).Value,
		}

		return existingUser, nil

	} else {
		// Otherwise return an empty UserData struct
		return UserData{}, nil
	}

}

func checkAndUpdateUserDetailsChanged(
	ctx context.Context,
	getItemInput *dynamodb.GetItemInput,
	existingUserData UserData,
	currentName string,
	currentPictureUrl string,
) (UserData, error) {

	// Check if name or pictureUrl changed, and if they did modify updateExpression
	updateNeeded := false
	updateExpression := "SET "
	expressionAttributeValues := map[string]types.AttributeValue{}

	// If name in existingUserData is different than the name received from Google add it to the expression
	if existingUserData.Name != currentName {
		// Modify update expression
		updateExpression += "Data.name = :name, "
		expressionAttributeValues[":name"] = &types.AttributeValueMemberS{Value: currentName}
		// Modify existingUserData
		existingUserData.Name = currentName
		// Set updateNeeded to true
		updateNeeded = true
	}

	// If pictureUrl in existingUserData is different than the pictureUrl received from Google add it to the expression
	if existingUserData.PictureUrl != currentPictureUrl {
		// Modify update expression
		updateExpression += "Data.pictureUrl = :pictureUrl, "
		expressionAttributeValues[":pictureUrl"] = &types.AttributeValueMemberS{Value: currentPictureUrl}
		// Modify existingUserData
		existingUserData.PictureUrl = currentPictureUrl
		// Set updateNeeded to true
		updateNeeded = true
	}

	// If updateNeeded was switched to true at any point, update using the updateExpression
	if updateNeeded {
		if err := updateUserProfile(ctx, getItemInput, updateExpression, expressionAttributeValues); err != nil {
			return UserData{}, err
		}
	}

	return existingUserData, nil

}

func updateUserProfile(
	ctx context.Context,
	getItemInput *dynamodb.GetItemInput,
	updateExpression string,
	expressionAttributeValues map[string]types.AttributeValue,
) error {
	updateExpression = updateExpression[:len(updateExpression)-2]
	_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       getItemInput.Key,
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	})
	if err != nil {
		return err
	}
	return nil
}
