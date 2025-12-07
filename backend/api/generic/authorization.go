package generic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AuthorizationError represents an authorization failure with HTTP status details
type AuthorizationError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

// NewForbiddenError creates a 403 Forbidden authorization error
func NewForbiddenError(message string) *AuthorizationError {
	return &AuthorizationError{
		StatusCode: 403,
		Code:       "FORBIDDEN",
		Message:    message,
	}
}

// NewNotFoundError creates a 404 Not Found error
func NewNotFoundError(message string) *AuthorizationError {
	return &AuthorizationError{
		StatusCode: 404,
		Code:       "NOT_FOUND",
		Message:    message,
	}
}

// CheckOrganizationMembership verifies if a user is a member of an organization
// Returns nil if authorized, error otherwise
func CheckOrganizationMembership(
	ctx context.Context,
	ddb *dynamodb.Client,
	tableName string,
	userID string,
	orgID string,
) error {
	if userID == "" {
		return NewForbiddenError("User does not have access to this organization")
	}
	if orgID == "" {
		return NewForbiddenError("User does not have access to this organization")
	}

	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", orgID)},
		"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
	}

	out, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		return fmt.Errorf("failed to check organization membership: %w", err)
	}

	if len(out.Item) == 0 {
		return NewForbiddenError("User does not have access to this organization")
	}

	return nil
}

// GetProjectOrganization fetches a project and extracts its organization ID
// Returns orgID and error
func GetProjectOrganization(
	ctx context.Context,
	ddb *dynamodb.Client,
	tableName string,
	projectID string,
) (string, error) {
	if projectID == "" {
		return "", NewNotFoundError("Project not found")
	}

	// Query for the project using GSI or scan with filter
	// Since projects have PK: ORG#<orgId> and SK: PROJECT#<projectId>,
	// we need to query by SK using a GSI or scan
	out, err := ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("SK-PK-index"),
		KeyConditionExpression: aws.String("SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch project: %w", err)
	}

	if len(out.Items) == 0 {
		return "", NewNotFoundError("Project not found")
	}

	// Extract organization ID from PK (format: "ORG#<orgId>")
	pkAttr, ok := out.Items[0]["PK"].(*types.AttributeValueMemberS)
	if !ok {
		return "", errors.New("invalid project item format: missing PK")
	}

	orgID := ExtractOrganizationID(pkAttr.Value)
	if orgID == "" {
		return "", errors.New("invalid project PK format")
	}

	return orgID, nil
}

// ExtractOrganizationID extracts the organization ID from a PK string
// PK format: "ORG#<orgId>"
// Returns the orgId without the "ORG#" prefix, or empty string if format is invalid
func ExtractOrganizationID(pk string) string {
	const prefix = "ORG#"
	if !strings.HasPrefix(pk, prefix) {
		return ""
	}
	return strings.TrimPrefix(pk, prefix)
}
