//go:build integration
// +build integration

package generic

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Integration tests for authorization. Requires DynamoDB Local on localhost:8000.
// Run: go test -tags=integration ./generic/...
// Start DynamoDB Local: docker run -p 8000:8000 amazon/dynamodb-local

const testTableName = "integration-test-table"

// TestData holds all test entities for integration tests
type TestData struct {
	// Organization 1: org-alpha with users user-alice and user-bob
	OrgAlpha     string
	UserAlice    string
	UserBob      string
	ProjectAlpha string
	DiagramAlpha string

	// Organization 2: org-beta with user user-charlie
	OrgBeta     string
	UserCharlie string
	ProjectBeta string
	DiagramBeta string

	// Unauthorized user (not member of any org)
	UserUnauthorized string
}

func getTestDDBClient(t *testing.T) (*dynamodb.Client, string) {
	endpoint := os.Getenv("DDB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return client, testTableName
}

// createTestTable creates the DynamoDB table with required indexes for integration tests
func createTestTable(ctx context.Context, ddb *dynamodb.Client, tableName string) error {
	// Check if table exists
	_, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		// Table exists, delete it first for clean state
		_, _ = ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		})
		// Wait for deletion
		time.Sleep(500 * time.Millisecond)
	}

	// Create table with GSI for SK-PK-index (required by GetProjectOrganization)
	_, err = ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("SK-PK-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("SK"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("PK"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be active
	time.Sleep(500 * time.Millisecond)
	return nil
}

// setupTestData creates all test data for integration tests
// Creates: organizations, users, projects, diagrams
func setupTestData(t *testing.T, ddb *dynamodb.Client, tableName string) *TestData {
	ctx := context.Background()

	data := &TestData{
		OrgAlpha:         "org-alpha",
		UserAlice:        "user-alice",
		UserBob:          "user-bob",
		ProjectAlpha:     "project-alpha",
		DiagramAlpha:     "diagram-alpha",
		OrgBeta:          "org-beta",
		UserCharlie:      "user-charlie",
		ProjectBeta:      "project-beta",
		DiagramBeta:      "diagram-beta",
		UserUnauthorized: "user-unauthorized",
	}

	// Create table
	if err := createTestTable(ctx, ddb, tableName); err != nil {
		t.Skipf("Could not create test table: %v", err)
	}

	// Organization Alpha memberships
	items := []map[string]types.AttributeValue{
		// User Alice is member of Org Alpha
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", data.OrgAlpha)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", data.UserAlice)},
			"Type": &types.AttributeValueMemberS{Value: "MEMBER"},
		},
		// User Bob is member of Org Alpha
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", data.OrgAlpha)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", data.UserBob)},
			"Type": &types.AttributeValueMemberS{Value: "MEMBER"},
		},
		// User Charlie is member of Org Beta
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", data.OrgBeta)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", data.UserCharlie)},
			"Type": &types.AttributeValueMemberS{Value: "MEMBER"},
		},
		// Project Alpha in Org Alpha
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", data.OrgAlpha)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", data.ProjectAlpha)},
			"Type": &types.AttributeValueMemberS{Value: "PROJECT"},
			"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":   &types.AttributeValueMemberS{Value: data.ProjectAlpha},
				"name": &types.AttributeValueMemberS{Value: "Project Alpha"},
			}},
		},
		// Project Beta in Org Beta
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", data.OrgBeta)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", data.ProjectBeta)},
			"Type": &types.AttributeValueMemberS{Value: "PROJECT"},
			"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":   &types.AttributeValueMemberS{Value: data.ProjectBeta},
				"name": &types.AttributeValueMemberS{Value: "Project Beta"},
			}},
		},
		// Diagram Alpha in Project Alpha
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", data.ProjectAlpha)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", data.DiagramAlpha)},
			"Type": &types.AttributeValueMemberS{Value: "DIAGRAM"},
			"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":        &types.AttributeValueMemberS{Value: data.DiagramAlpha},
				"name":      &types.AttributeValueMemberS{Value: "Diagram Alpha"},
				"createdBy": &types.AttributeValueMemberS{Value: data.UserAlice},
				"createdAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			}},
		},
		// Diagram Beta in Project Beta
		{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", data.ProjectBeta)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", data.DiagramBeta)},
			"Type": &types.AttributeValueMemberS{Value: "DIAGRAM"},
			"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":        &types.AttributeValueMemberS{Value: data.DiagramBeta},
				"name":      &types.AttributeValueMemberS{Value: "Diagram Beta"},
				"createdBy": &types.AttributeValueMemberS{Value: data.UserCharlie},
				"createdAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			}},
		},
	}

	for _, item := range items {
		_, err := ddb.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			t.Skipf("Could not setup test data: %v", err)
		}
	}

	return data
}

// cleanupTestTable deletes the test table
func cleanupTestTable(ctx context.Context, ddb *dynamodb.Client, tableName string) {
	_, _ = ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
}

// =============================================================================
// Integration Tests for CheckOrganizationMembership
// =============================================================================

func TestCheckOrganizationMembership_Integration(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	t.Run("authorized user can access their organization", func(t *testing.T) {
		// Alice is member of OrgAlpha
		err := CheckOrganizationMembership(ctx, ddb, tableName, data.UserAlice, data.OrgAlpha)
		if err != nil {
			t.Errorf("Expected no error for authorized user, got: %v", err)
		}
	})

	t.Run("unauthorized user gets 403 Forbidden", func(t *testing.T) {
		// Charlie is NOT member of OrgAlpha
		err := CheckOrganizationMembership(ctx, ddb, tableName, data.UserCharlie, data.OrgAlpha)
		if err == nil {
			t.Error("Expected error for unauthorized user")
			return
		}

		authErr, ok := err.(*AuthorizationError)
		if !ok {
			t.Errorf("Expected AuthorizationError, got: %T", err)
			return
		}

		if authErr.StatusCode != 403 {
			t.Errorf("Expected 403, got %d", authErr.StatusCode)
		}

		if authErr.Code != "FORBIDDEN" {
			t.Errorf("Expected code FORBIDDEN, got %s", authErr.Code)
		}

		if authErr.Message == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("non-existent user gets 403 Forbidden", func(t *testing.T) {
		err := CheckOrganizationMembership(ctx, ddb, tableName, data.UserUnauthorized, data.OrgAlpha)
		if err == nil {
			t.Error("Expected error for non-existent user")
			return
		}

		authErr, ok := err.(*AuthorizationError)
		if !ok {
			t.Errorf("Expected AuthorizationError, got: %T", err)
			return
		}

		if authErr.StatusCode != 403 {
			t.Errorf("Expected 403, got %d", authErr.StatusCode)
		}
	})
}

// =============================================================================
// Integration Tests for GetProjectOrganization
// =============================================================================

func TestGetProjectOrganization_Integration(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	t.Run("existing project returns correct organization ID", func(t *testing.T) {
		orgID, err := GetProjectOrganization(ctx, ddb, tableName, data.ProjectAlpha)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
			return
		}

		if orgID != data.OrgAlpha {
			t.Errorf("Expected orgID %s, got %s", data.OrgAlpha, orgID)
		}
	})

	t.Run("non-existent project returns 404 Not Found", func(t *testing.T) {
		_, err := GetProjectOrganization(ctx, ddb, tableName, "non-existent-project")
		if err == nil {
			t.Error("Expected error for non-existent project")
			return
		}

		authErr, ok := err.(*AuthorizationError)
		if !ok {
			t.Errorf("Expected AuthorizationError, got: %T", err)
			return
		}

		if authErr.StatusCode != 404 {
			t.Errorf("Expected 404, got %d", authErr.StatusCode)
		}

		if authErr.Code != "NOT_FOUND" {
			t.Errorf("Expected code NOT_FOUND, got %s", authErr.Code)
		}
	})
}

// =============================================================================
// End-to-End Authorization Flow Tests
// =============================================================================

// TestAuthorizationFlow_MultipleUsersMultipleOrgs tests auth with multiple users and orgs
func TestAuthorizationFlow_MultipleUsersMultipleOrgs(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	testCases := []struct {
		name           string
		userID         string
		projectID      string
		expectOrgID    string
		expectAuthErr  bool
		expectNotFound bool
	}{
		{
			name:        "Alice can access ProjectAlpha (member of OrgAlpha)",
			userID:      data.UserAlice,
			projectID:   data.ProjectAlpha,
			expectOrgID: data.OrgAlpha,
		},
		{
			name:        "Bob can access ProjectAlpha (member of OrgAlpha)",
			userID:      data.UserBob,
			projectID:   data.ProjectAlpha,
			expectOrgID: data.OrgAlpha,
		},
		{
			name:          "Charlie cannot access ProjectAlpha (not member of OrgAlpha)",
			userID:        data.UserCharlie,
			projectID:     data.ProjectAlpha,
			expectAuthErr: true,
		},
		{
			name:        "Charlie can access ProjectBeta (member of OrgBeta)",
			userID:      data.UserCharlie,
			projectID:   data.ProjectBeta,
			expectOrgID: data.OrgBeta,
		},
		{
			name:          "Alice cannot access ProjectBeta (not member of OrgBeta)",
			userID:        data.UserAlice,
			projectID:     data.ProjectBeta,
			expectAuthErr: true,
		},
		{
			name:           "Any user gets 404 for non-existent project",
			userID:         data.UserAlice,
			projectID:      "non-existent-project",
			expectNotFound: true,
		},
		{
			name:          "Unauthorized user cannot access any project",
			userID:        data.UserUnauthorized,
			projectID:     data.ProjectAlpha,
			expectAuthErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Get project's organization
			orgID, err := GetProjectOrganization(ctx, ddb, tableName, tc.projectID)

			if tc.expectNotFound {
				if err == nil {
					t.Error("Expected 404 error for non-existent project")
					return
				}
				authErr, ok := err.(*AuthorizationError)
				if !ok || authErr.StatusCode != 404 {
					t.Errorf("Expected 404 AuthorizationError, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error getting project organization: %v", err)
				return
			}

			// Step 2: Check organization membership
			err = CheckOrganizationMembership(ctx, ddb, tableName, tc.userID, orgID)

			if tc.expectAuthErr {
				if err == nil {
					t.Error("Expected 403 error for unauthorized user")
					return
				}
				authErr, ok := err.(*AuthorizationError)
				if !ok {
					t.Errorf("Expected AuthorizationError, got: %T", err)
					return
				}
				if authErr.StatusCode != 403 {
					t.Errorf("Expected 403, got %d", authErr.StatusCode)
				}
				if authErr.Code != "FORBIDDEN" {
					t.Errorf("Expected code FORBIDDEN, got %s", authErr.Code)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected authorization error: %v", err)
				return
			}

			if orgID != tc.expectOrgID {
				t.Errorf("Expected orgID %s, got %s", tc.expectOrgID, orgID)
			}
		})
	}
}

// TestErrorResponseConsistency verifies error responses follow a consistent format
func TestErrorResponseConsistency(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	t.Run("403 error has consistent format", func(t *testing.T) {
		err := CheckOrganizationMembership(ctx, ddb, tableName, data.UserUnauthorized, data.OrgAlpha)
		authErr, ok := err.(*AuthorizationError)
		if !ok {
			t.Fatalf("Expected AuthorizationError, got: %T", err)
		}

		// Verify all required fields are present
		if authErr.StatusCode != 403 {
			t.Errorf("StatusCode: expected 403, got %d", authErr.StatusCode)
		}
		if authErr.Code != "FORBIDDEN" {
			t.Errorf("Code: expected FORBIDDEN, got %s", authErr.Code)
		}
		if authErr.Message == "" {
			t.Error("Message: expected non-empty string")
		}
	})

	t.Run("404 error has consistent format", func(t *testing.T) {
		_, err := GetProjectOrganization(ctx, ddb, tableName, "non-existent")
		authErr, ok := err.(*AuthorizationError)
		if !ok {
			t.Fatalf("Expected AuthorizationError, got: %T", err)
		}

		// Verify all required fields are present
		if authErr.StatusCode != 404 {
			t.Errorf("StatusCode: expected 404, got %d", authErr.StatusCode)
		}
		if authErr.Code != "NOT_FOUND" {
			t.Errorf("Code: expected NOT_FOUND, got %s", authErr.Code)
		}
		if authErr.Message == "" {
			t.Error("Message: expected non-empty string")
		}
	})
}

// TestCrossOrganizationAccess verifies that users cannot access resources across organizations
func TestCrossOrganizationAccess(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	// Test that OrgAlpha members cannot access OrgBeta resources
	t.Run("OrgAlpha members cannot access OrgBeta", func(t *testing.T) {
		// Get OrgBeta's project
		orgID, err := GetProjectOrganization(ctx, ddb, tableName, data.ProjectBeta)
		if err != nil {
			t.Fatalf("Failed to get project organization: %v", err)
		}

		// Alice (OrgAlpha member) should not have access
		err = CheckOrganizationMembership(ctx, ddb, tableName, data.UserAlice, orgID)
		if err == nil {
			t.Error("Expected Alice to be denied access to OrgBeta")
		}

		// Bob (OrgAlpha member) should not have access
		err = CheckOrganizationMembership(ctx, ddb, tableName, data.UserBob, orgID)
		if err == nil {
			t.Error("Expected Bob to be denied access to OrgBeta")
		}
	})

	// Test that OrgBeta members cannot access OrgAlpha resources
	t.Run("OrgBeta members cannot access OrgAlpha", func(t *testing.T) {
		// Get OrgAlpha's project
		orgID, err := GetProjectOrganization(ctx, ddb, tableName, data.ProjectAlpha)
		if err != nil {
			t.Fatalf("Failed to get project organization: %v", err)
		}

		// Charlie (OrgBeta member) should not have access
		err = CheckOrganizationMembership(ctx, ddb, tableName, data.UserCharlie, orgID)
		if err == nil {
			t.Error("Expected Charlie to be denied access to OrgAlpha")
		}
	})
}

// =============================================================================
// Diagram CRUD Simulation Tests
// These tests simulate the exact authorization flow used by diagram handlers
// =============================================================================

// SimulateDiagramCreate simulates the create diagram handler's authorization flow
// Returns: diagramID on success, or error
func SimulateDiagramCreate(ctx context.Context, ddb *dynamodb.Client, tableName, userID, projectID, diagramName string) (string, error) {
	// Step 1: Get project's organization (same as handler)
	orgID, err := GetProjectOrganization(ctx, ddb, tableName, projectID)
	if err != nil {
		return "", err
	}

	// Step 2: Check organization membership (same as handler)
	if err := CheckOrganizationMembership(ctx, ddb, tableName, userID, orgID); err != nil {
		return "", err
	}

	// Step 3: Create diagram (simulated - just insert into DynamoDB)
	diagramID := fmt.Sprintf("diagram-%d", time.Now().UnixNano())
	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
			"Type": &types.AttributeValueMemberS{Value: "DIAGRAM"},
			"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":        &types.AttributeValueMemberS{Value: diagramID},
				"name":      &types.AttributeValueMemberS{Value: diagramName},
				"createdBy": &types.AttributeValueMemberS{Value: userID},
				"createdAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create diagram: %w", err)
	}

	return diagramID, nil
}

// SimulateDiagramGet simulates the get diagram handler's authorization flow
func SimulateDiagramGet(ctx context.Context, ddb *dynamodb.Client, tableName, userID, projectID, diagramID string) (map[string]types.AttributeValue, error) {
	// Step 1: Get project's organization
	orgID, err := GetProjectOrganization(ctx, ddb, tableName, projectID)
	if err != nil {
		return nil, err
	}

	// Step 2: Check organization membership
	if err := CheckOrganizationMembership(ctx, ddb, tableName, userID, orgID); err != nil {
		return nil, err
	}

	// Step 3: Get diagram
	out, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get diagram: %w", err)
	}

	if out.Item == nil {
		return nil, NewNotFoundError("Diagram not found")
	}

	return out.Item, nil
}

// SimulateDiagramUpdate simulates the update diagram handler's authorization flow
func SimulateDiagramUpdate(ctx context.Context, ddb *dynamodb.Client, tableName, userID, projectID, diagramID, newName string) error {
	// Step 1: Get project's organization
	orgID, err := GetProjectOrganization(ctx, ddb, tableName, projectID)
	if err != nil {
		return err
	}

	// Step 2: Check organization membership
	if err := CheckOrganizationMembership(ctx, ddb, tableName, userID, orgID); err != nil {
		return err
	}

	// Step 3: Update diagram
	_, err = ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
		UpdateExpression: aws.String("SET #Data.#name = :name, #Data.#updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#Data":      "Data",
			"#name":      "name",
			"#updatedAt": "updatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name":      &types.AttributeValueMemberS{Value: newName},
			":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		return fmt.Errorf("failed to update diagram: %w", err)
	}

	return nil
}

// SimulateDiagramDelete simulates the delete diagram handler's authorization flow
func SimulateDiagramDelete(ctx context.Context, ddb *dynamodb.Client, tableName, userID, projectID, diagramID string) error {
	// Step 1: Get project's organization
	orgID, err := GetProjectOrganization(ctx, ddb, tableName, projectID)
	if err != nil {
		return err
	}

	// Step 2: Check organization membership
	if err := CheckOrganizationMembership(ctx, ddb, tableName, userID, orgID); err != nil {
		return err
	}

	// Step 3: Delete diagram
	_, err = ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PROJECT#%s", projectID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("DIAGRAM#%s", diagramID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		return fmt.Errorf("failed to delete diagram: %w", err)
	}

	return nil
}

// TestDiagramCRUD_AuthorizedUser tests full CRUD operations for an authorized user
func TestDiagramCRUD_AuthorizedUser(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	t.Run("CREATE: authorized user can create diagram", func(t *testing.T) {
		diagramID, err := SimulateDiagramCreate(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, "New Diagram")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if diagramID == "" {
			t.Error("Expected non-empty diagram ID")
		}
		t.Logf("Created diagram: %s", diagramID)
	})

	t.Run("GET: authorized user can read diagram", func(t *testing.T) {
		item, err := SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, data.DiagramAlpha)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if item == nil {
			t.Error("Expected diagram data")
		}
	})

	t.Run("UPDATE: authorized user can update diagram", func(t *testing.T) {
		err := SimulateDiagramUpdate(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, data.DiagramAlpha, "Updated Name")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		// Verify update
		item, _ := SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, data.DiagramAlpha)
		if item == nil {
			t.Fatal("Expected diagram data")
		}

		if dataAttr, ok := item["Data"].(*types.AttributeValueMemberM); ok {
			if nameAttr, ok := dataAttr.Value["name"].(*types.AttributeValueMemberS); ok {
				if nameAttr.Value != "Updated Name" {
					t.Errorf("Expected 'Updated Name', got '%s'", nameAttr.Value)
				}
			} else {
				t.Errorf("Name attribute not found or wrong type: %T", dataAttr.Value["name"])
			}
		} else {
			t.Errorf("Data attribute not found or wrong type: %T", item["Data"])
		}
	})

	t.Run("DELETE: authorized user can delete diagram", func(t *testing.T) {
		// Create a diagram to delete
		diagramID, _ := SimulateDiagramCreate(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, "To Delete")

		err := SimulateDiagramDelete(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, diagramID)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		// Verify deletion
		_, err = SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, diagramID)
		if err == nil {
			t.Error("Expected 404 error for deleted diagram")
		}
	})
}

// TestDiagramCRUD_UnauthorizedUser tests that unauthorized users are blocked from all operations
func TestDiagramCRUD_UnauthorizedUser(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	// Charlie (OrgBeta member) trying to access OrgAlpha's project
	t.Run("CREATE: unauthorized user gets 403", func(t *testing.T) {
		_, err := SimulateDiagramCreate(ctx, ddb, tableName, data.UserCharlie, data.ProjectAlpha, "Hacked Diagram")
		if err == nil {
			t.Fatal("Expected 403 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 403 {
			t.Errorf("Expected 403 AuthorizationError, got: %v", err)
		}
	})

	t.Run("GET: unauthorized user gets 403", func(t *testing.T) {
		_, err := SimulateDiagramGet(ctx, ddb, tableName, data.UserCharlie, data.ProjectAlpha, data.DiagramAlpha)
		if err == nil {
			t.Fatal("Expected 403 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 403 {
			t.Errorf("Expected 403 AuthorizationError, got: %v", err)
		}
	})

	t.Run("UPDATE: unauthorized user gets 403", func(t *testing.T) {
		err := SimulateDiagramUpdate(ctx, ddb, tableName, data.UserCharlie, data.ProjectAlpha, data.DiagramAlpha, "Hacked Name")
		if err == nil {
			t.Fatal("Expected 403 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 403 {
			t.Errorf("Expected 403 AuthorizationError, got: %v", err)
		}
	})

	t.Run("DELETE: unauthorized user gets 403", func(t *testing.T) {
		err := SimulateDiagramDelete(ctx, ddb, tableName, data.UserCharlie, data.ProjectAlpha, data.DiagramAlpha)
		if err == nil {
			t.Fatal("Expected 403 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 403 {
			t.Errorf("Expected 403 AuthorizationError, got: %v", err)
		}
	})

	t.Run("Data integrity: diagram unchanged after unauthorized update attempt", func(t *testing.T) {
		// Get original diagram
		originalItem, _ := SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, data.DiagramAlpha)

		// Attempt unauthorized update
		_ = SimulateDiagramUpdate(ctx, ddb, tableName, data.UserCharlie, data.ProjectAlpha, data.DiagramAlpha, "Hacked Name")

		// Verify diagram unchanged
		currentItem, _ := SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, data.ProjectAlpha, data.DiagramAlpha)

		originalData := originalItem["Data"].(*types.AttributeValueMemberM).Value
		currentData := currentItem["Data"].(*types.AttributeValueMemberM).Value

		originalName := originalData["name"].(*types.AttributeValueMemberS).Value
		currentName := currentData["name"].(*types.AttributeValueMemberS).Value

		if originalName != currentName {
			t.Errorf("Data integrity violated: name changed from '%s' to '%s'", originalName, currentName)
		}
	})
}

// TestDiagramCRUD_NonExistentProject tests 404 responses for non-existent projects
func TestDiagramCRUD_NonExistentProject(t *testing.T) {
	ddb, tableName := getTestDDBClient(t)
	ctx := context.Background()
	data := setupTestData(t, ddb, tableName)
	defer cleanupTestTable(ctx, ddb, tableName)

	nonExistentProject := "non-existent-project"

	t.Run("CREATE: non-existent project returns 404", func(t *testing.T) {
		_, err := SimulateDiagramCreate(ctx, ddb, tableName, data.UserAlice, nonExistentProject, "Test")
		if err == nil {
			t.Fatal("Expected 404 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 404 {
			t.Errorf("Expected 404 AuthorizationError, got: %v", err)
		}
	})

	t.Run("GET: non-existent project returns 404", func(t *testing.T) {
		_, err := SimulateDiagramGet(ctx, ddb, tableName, data.UserAlice, nonExistentProject, "any-diagram")
		if err == nil {
			t.Fatal("Expected 404 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 404 {
			t.Errorf("Expected 404 AuthorizationError, got: %v", err)
		}
	})

	t.Run("UPDATE: non-existent project returns 404", func(t *testing.T) {
		err := SimulateDiagramUpdate(ctx, ddb, tableName, data.UserAlice, nonExistentProject, "any-diagram", "New Name")
		if err == nil {
			t.Fatal("Expected 404 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 404 {
			t.Errorf("Expected 404 AuthorizationError, got: %v", err)
		}
	})

	t.Run("DELETE: non-existent project returns 404", func(t *testing.T) {
		err := SimulateDiagramDelete(ctx, ddb, tableName, data.UserAlice, nonExistentProject, "any-diagram")
		if err == nil {
			t.Fatal("Expected 404 error")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 404 {
			t.Errorf("Expected 404 AuthorizationError, got: %v", err)
		}
	})
}
