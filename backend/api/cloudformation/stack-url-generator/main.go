package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Alberta2514640/clutter/backend/api/cloudformation/stack-url-generator/internal"
	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// The authorizer context retains data about the user (extracted from the JWT provided in the Authorization header)
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from authorizer context", "error": err.Error()},
		)
	}
	userId := userData.Id

	// Read path and query parameters
	// Required: organizationId (Path Param)
	// Required: accountName    (Query Param)
	// Optional: region         (Query Param)
	organizationId := request.PathParameters["organizationId"]

	accountName := request.QueryStringParameters["accountName"]
	region := request.QueryStringParameters["region"]

	// Check to see if organization ID is not passed in
	if organizationId == "" {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "no 'organizationId' path parameter inputted."},
		)
	}

	// Check to see if organization ID passed in is valid UUID
	isValidUuid := generic.IsValidUuid(organizationId)
	if !isValidUuid {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "inputted 'organizationId' path parameter is not a valid UUID."},
		)
	}

	// Validate accountName
	if err := internal.ValidateAccountName(accountName); err != nil {
		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": "something went wrong while validating the 'accountName' query parameter", "error": err.Error()},
		)
	}

	// Ensure region is supported and valid
	// Set default region if not passed in
	if region == "" {
		region = generic.DefaultRegion
	}
	if !generic.IsRegionSupported(region) {

		var supportedRegionsSlice []string

		for key := range generic.SupportedRegions {
			supportedRegionsSlice = append(supportedRegionsSlice, key)
		}

		return generic.Response(
			http.StatusBadRequest,
			generic.Json{"message": fmt.Sprintf("invalid AWS region passed in: %s", region), "options": supportedRegionsSlice},
		)
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to connect to PostgreSQL database",
			"error":   err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	// Source: ChatGPT
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, organizationId)
	if err != nil {
		if authErr, ok := err.(*generic.AuthorizationError); ok {
			// Handle 404 Not Found
			if authErr.StatusCode == 404 {
				return generic.Response(http.StatusNotFound, generic.Json{
					"error": authErr.Message,
				})
			}

			// Handle 403 Forbidden
			if authErr.StatusCode == 403 {
				return generic.Response(http.StatusForbidden, generic.Json{
					"error": authErr.Message,
				})
			}
		}

		// Fallback for unexpected errors
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "internal server error",
			"error":   err.Error(),
		})
	}

	// Begin URL generation

	// Query organization name from ID
	var organizationName string
	query := `SELECT name FROM organizations WHERE id = $1`
	err = conn.QueryRow(ctx, query, organizationId).Scan(&organizationName)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to query organization name",
			"error":   err.Error(),
		})
	}

	// Remove spaces and single quotes for use in stack name
	safeOrgName := strings.ReplaceAll(organizationName, " ", "")
	safeOrgName = strings.ReplaceAll(safeOrgName, "'", "")

	// Define stack variables
	templateUrlEnvVarKey := "CLOUDFORMATION_TEMPLATE_URL"
	clutterAccountIdEnvVarKey := "CLUTTER_ACCOUNT_ID"

	templateUrl := os.Getenv(templateUrlEnvVarKey)
	clutterAccountId := os.Getenv(clutterAccountIdEnvVarKey)

	// Check to see if required env vars are provided
	var missingEnvVars []string
	var envVarMissing bool = false

	if templateUrl == "" {
		envVarMissing = true
		missingEnvVars = append(missingEnvVars, templateUrlEnvVarKey)
	}

	if clutterAccountId == "" {
		envVarMissing = true
		missingEnvVars = append(missingEnvVars, clutterAccountIdEnvVarKey)
	}

	if envVarMissing {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "the following required environment variables are missing on the Lambda function",
			"missing": missingEnvVars,
		})
	}

	uniqueId := internal.RandomID(8)
	// Stack name has format of: Clutter-<organizationName>-<accountName>-TerraformDeployerRoleStack-<uniqueId>
	stackName := fmt.Sprintf("Clutter-%s-%s-TerraformDeployerRoleStack-%s", safeOrgName, accountName, uniqueId)
	externalId := uuid.NewString()

	cloudformationStackQueryParams := map[string]string{
		"templateURL":      templateUrl,
		"stackName":        stackName,
		"param_AccountId":  clutterAccountId,
		"param_ExternalId": externalId,
		"param_UniqueId":   uniqueId,
	}

	fullUrl := internal.BuildCloudFormationURL(region, cloudformationStackQueryParams)

	// Insert AWS IAM Role details into DB
	insertQuery := `
	INSERT INTO public.aws_account_access_roles
	(id, organization_id, unique_id, account_name, external_id, created_by)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	recordId := uuid.New()

	_, err = conn.Exec(ctx, insertQuery,
		recordId,
		organizationId,
		uniqueId,
		accountName,
		externalId,
		userId,
	)
	if err != nil {
		// Check if this is a unique constraint violation
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return generic.Response(http.StatusConflict, generic.Json{
				"message": "account with this name already exists for the organization",
				"error":   pgErr.Message,
			})
		}

		// Any other error
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"message": "failed to insert aws_account_access_roles record",
			"error":   err.Error(),
		})
	}

	return generic.Response(
		http.StatusOK,
		generic.Json{
			"message": "The URL was generated successfully and a partial client AWS Account Role record has been created in the DB. Client must now provide the Role ARN once the CloudFormation stack is successfully created to complete the account access setup.",
			"data": generic.Json{
				"account_id": recordId,
				"url":        fullUrl,
			},
		},
	)

}
