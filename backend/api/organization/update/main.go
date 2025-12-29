package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/organization/update/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {

	lambda.Start(handler)

}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get org ID from path paremeter
	orgId := request.PathParameters["organizationId"]
	// Check if org ID is a valid UUID
	isValidUuid := generic.IsValidUuid(orgId)
	if !isValidUuid {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"message": "inputted 'organizationId' path parameter is not a valid UUID.",
		})
	}

	// The authorizer context retains data about the user extracted from the JWT inside the authorizer Lambda function
	userData, err := generic.GetUserDataFromAuthorizerContext(request.RequestContext.Authorizer)
	if err != nil {
		return generic.Response(
			http.StatusInternalServerError,
			generic.Json{"message": "failed to retrieve user data from autorizer context", "error": err.Error()},
		)
	}
	// Get user ID from user authorizer data
	userId := userData.Id

	// Process body from API Gateway request
	var body generic.OrgRequestBody
	err = json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{"message": "Bad Request", "error": err.Error()})
	}
	// Get org name and/or description from request body
	newOrgName := body.OrganizationName
	newDescription := body.Description

	// Connect to PostgreSQL
	ctx := context.Background()
	conn, err := generic.PsqlConnect()
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to connect to PostgreSQL database",
			"message": err.Error(),
		})
	}
	defer conn.Close(ctx)

	// Check membership of user to organization
	// Source: ChatGPT
	err = generic.CheckOrganizationMembershipPSQL(ctx, conn, userId, orgId)
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

	// Fetch current org data
	var currentOrgName, currentDescription string

	queryCurrentData := `
		SELECT name, description
		FROM organizations
		WHERE id = $1
	`

	row := conn.QueryRow(ctx, queryCurrentData, orgId)
	if err := row.Scan(&currentOrgName, &currentDescription); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to fetch current organization data",
			"message": err.Error(),
		})
	}

	// Check if anything changed
	nameUnchanged := newOrgName == "" || newOrgName == currentOrgName
	descriptionUnchanged := newDescription == "" || newDescription == currentDescription

	if nameUnchanged && descriptionUnchanged {
		return generic.Response(http.StatusOK, generic.Json{
			"message": "no changes detected, organization not updated",
		})
	}

	// Begin transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to start transaction",
			"message": err.Error(),
		})
	}
	defer tx.Rollback(ctx)

	// Update organization name only if new and current name don't match
	if newOrgName != "" && newOrgName != currentOrgName {

		err = internal.UpdateOrgName(tx, ctx, newOrgName, orgId)

		if generic.IsUniqueViolation(err) {
			return generic.Response(http.StatusConflict, generic.Json{
				"error":   "organization name already exists for this user",
				"message": err.Error(),
			})
		} else if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"error":   "failed to update organization name",
				"message": err.Error(),
			})
		}

	}

	// Update organization description only if new and current description don't match
	if newDescription != "" && newDescription != currentDescription {

		err = internal.UpdateOrgDescription(tx, ctx, newDescription, orgId)

		if err != nil {
			return generic.Response(http.StatusInternalServerError, generic.Json{
				"error":   "failed to update organization description",
				"message": err.Error(),
			})
		}

	}

	// Query all data for updated org
	querySingleOrg := fmt.Sprintf(
		generic.QueryOrgData,
		"WHERE o.id = $1 AND om.member_id = $2",
	)

	// Initialize empty orgData struct to populate in scan
	var orgData generic.OrgOverviewData
	// Initialize variable to hold projects json as bytes
	var projectsJson []byte

	// Get data of new updated organization for use by the frontend
	row = tx.QueryRow(ctx, querySingleOrg, orgId, userId)
	err = row.Scan(
		&orgData.Id,
		&orgData.CreatedBy,
		&orgData.Name,
		&orgData.Description,
		&orgData.CreatedAt,
		&orgData.TotalMembers,
		&orgData.Members,
		&orgData.TotalProjects,
		&projectsJson,
		&orgData.TotalDiagrams,
	)
	if err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to fetch organization overview",
			"message": err.Error(),
		})
	}

	// Unmarshal projects json as bytes into Projects slice
	if err := json.Unmarshal(projectsJson, &orgData.Projects); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to parse projects data",
			"message": err.Error(),
		})
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return generic.Response(http.StatusInternalServerError, generic.Json{
			"error":   "failed to commit organization update transaction",
			"message": err.Error(),
		})
	}

	// Organization update process completed successfully and new data can be returned
	return generic.Response(http.StatusOK, generic.Json{
		"message": "organization updated successfully",
		"data":    orgData,
	})
}
