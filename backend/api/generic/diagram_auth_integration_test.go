package generic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDiagramAuthorizationFlow_Integration tests the complete authorization flow for diagrams
// This test requires a PostgreSQL database connection
func TestDiagramAuthorizationFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we can connect to PostgreSQL
	conn, err := PsqlConnect()
	if err != nil {
		t.Skipf("Skipping integration test - cannot connect to PostgreSQL: %v", err)
	}
	defer conn.Close(context.Background())

	ctx := context.Background()

	// Setup: Create test data
	userID := uuid.NewString()
	otherUserID := uuid.NewString()
	orgID := uuid.NewString()
	projectID := uuid.NewString()
	diagramID := uuid.NewString()

	// Cleanup function
	cleanup := func() {
		conn.Exec(ctx, "DELETE FROM diagrams WHERE id = $1", diagramID)
		conn.Exec(ctx, "DELETE FROM projects WHERE id = $1", projectID)
		conn.Exec(ctx, "DELETE FROM organization_members WHERE member_id IN ($1, $2)", userID, otherUserID)
		conn.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)
		conn.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", userID, otherUserID)
	}
	defer cleanup()

	t.Run("Setup test data", func(t *testing.T) {
		// Create users
		_, err := conn.Exec(ctx, `
			INSERT INTO users (id, email, full_name, picture_url, account_created_on)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, "test@example.com", "Test User", "https://example.com/pic.jpg", time.Now())
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		_, err = conn.Exec(ctx, `
			INSERT INTO users (id, email, full_name, picture_url, account_created_on)
			VALUES ($1, $2, $3, $4, $5)
		`, otherUserID, "other@example.com", "Other User", "https://example.com/pic2.jpg", time.Now())
		if err != nil {
			t.Fatalf("Failed to create other user: %v", err)
		}

		// Create organization
		_, err = conn.Exec(ctx, `
			INSERT INTO organizations (id, created_by, name, description, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, orgID, userID, "Test Org", "Test organization", time.Now())
		if err != nil {
			t.Fatalf("Failed to create organization: %v", err)
		}

		// Add user to organization
		_, err = conn.Exec(ctx, `
			INSERT INTO organization_members (organization_id, member_id, joined_at)
			VALUES ($1, $2, $3)
		`, orgID, userID, time.Now())
		if err != nil {
			t.Fatalf("Failed to add user to organization: %v", err)
		}

		// Create project
		_, err = conn.Exec(ctx, `
			INSERT INTO projects (id, organization_id, created_by, name, description, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, projectID, orgID, userID, "Test Project", "Test project", time.Now())
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Create diagram
		_, err = conn.Exec(ctx, `
			INSERT INTO diagrams (id, project_id, created_by, name, data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, diagramID, projectID, userID, "Test Diagram", "{}", time.Now())
		if err != nil {
			t.Fatalf("Failed to create diagram: %v", err)
		}
	})

	t.Run("GetDiagramProjectPSQL returns correct project", func(t *testing.T) {
		retrievedProjectID, err := GetDiagramProjectPSQL(ctx, conn, diagramID)
		if err != nil {
			t.Fatalf("GetDiagramProjectPSQL failed: %v", err)
		}
		if retrievedProjectID != projectID {
			t.Errorf("Expected project ID %s, got %s", projectID, retrievedProjectID)
		}
	})

	t.Run("GetDiagramProjectPSQL returns error for non-existent diagram", func(t *testing.T) {
		_, err := GetDiagramProjectPSQL(ctx, conn, uuid.NewString())
		if err == nil {
			t.Error("Expected error for non-existent diagram")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 404 {
			t.Errorf("Expected 404 error, got: %v", err)
		}
	})

	t.Run("GetProjectOrganizationPSQL returns correct organization", func(t *testing.T) {
		retrievedOrgID, err := GetProjectOrganizationPSQL(ctx, conn, projectID)
		if err != nil {
			t.Fatalf("GetProjectOrganizationPSQL failed: %v", err)
		}
		if retrievedOrgID != orgID {
			t.Errorf("Expected org ID %s, got %s", orgID, retrievedOrgID)
		}
	})

	t.Run("CheckOrganizationMembershipPSQL allows authorized user", func(t *testing.T) {
		err := CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID)
		if err != nil {
			t.Errorf("Authorized user should have access, got error: %v", err)
		}
	})

	t.Run("CheckOrganizationMembershipPSQL blocks unauthorized user", func(t *testing.T) {
		err := CheckOrganizationMembershipPSQL(ctx, conn, otherUserID, orgID)
		if err == nil {
			t.Error("Unauthorized user should not have access")
		}
		authErr, ok := err.(*AuthorizationError)
		if !ok || authErr.StatusCode != 403 {
			t.Errorf("Expected 403 error, got: %v", err)
		}
	})

	t.Run("Complete authorization chain for diagram access", func(t *testing.T) {
		// Simulate the full authorization flow used in diagram endpoints

		// Step 1: Get diagram's project
		retrievedProjectID, err := GetDiagramProjectPSQL(ctx, conn, diagramID)
		if err != nil {
			t.Fatalf("Step 1 failed: %v", err)
		}

		// Step 2: Get project's organization
		retrievedOrgID, err := GetProjectOrganizationPSQL(ctx, conn, retrievedProjectID)
		if err != nil {
			t.Fatalf("Step 2 failed: %v", err)
		}

		// Step 3: Check organization membership for authorized user
		err = CheckOrganizationMembershipPSQL(ctx, conn, userID, retrievedOrgID)
		if err != nil {
			t.Errorf("Authorized user should pass full chain, got error: %v", err)
		}

		// Step 4: Check organization membership for unauthorized user
		err = CheckOrganizationMembershipPSQL(ctx, conn, otherUserID, retrievedOrgID)
		if err == nil {
			t.Error("Unauthorized user should fail authorization chain")
		}
	})

	t.Run("Verify foreign key relationships", func(t *testing.T) {
		// Test CASCADE delete: deleting project should delete diagram
		testProjectID := uuid.NewString()
		testDiagramID := uuid.NewString()

		_, err := conn.Exec(ctx, `
			INSERT INTO projects (id, organization_id, created_by, name, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, testProjectID, orgID, userID, "Test FK Project", time.Now())
		if err != nil {
			t.Fatalf("Failed to create test project: %v", err)
		}

		_, err = conn.Exec(ctx, `
			INSERT INTO diagrams (id, project_id, created_by, name, data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, testDiagramID, testProjectID, userID, "Test FK Diagram", "{}", time.Now())
		if err != nil {
			t.Fatalf("Failed to create test diagram: %v", err)
		}

		// Delete the project
		_, err = conn.Exec(ctx, "DELETE FROM projects WHERE id = $1", testProjectID)
		if err != nil {
			t.Fatalf("Failed to delete project: %v", err)
		}

		// Verify diagram was cascade deleted
		var count int
		err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM diagrams WHERE id = $1", testDiagramID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check diagram count: %v", err)
		}
		if count != 0 {
			t.Error("Diagram should have been cascade deleted with project")
		}
	})

	t.Run("Verify unique constraints", func(t *testing.T) {
		// Try to create diagram with duplicate name in same project
		duplicateDiagramID := uuid.NewString()
		_, err := conn.Exec(ctx, `
			INSERT INTO diagrams (id, project_id, created_by, name, data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, duplicateDiagramID, projectID, userID, "Test Diagram", "{}", time.Now())

		if err == nil {
			t.Error("Should not allow duplicate diagram name in same project")
			conn.Exec(ctx, "DELETE FROM diagrams WHERE id = $1", duplicateDiagramID)
		}
	})

	t.Run("Verify nullable latest_update_by field", func(t *testing.T) {
		// Check that latest_update_by is NULL initially
		var latestUpdateBy *string
		err := conn.QueryRow(ctx, `
			SELECT latest_update_by FROM diagrams WHERE id = $1
		`, diagramID).Scan(&latestUpdateBy)
		if err != nil {
			t.Fatalf("Failed to query diagram: %v", err)
		}
		if latestUpdateBy != nil {
			t.Error("latest_update_by should be NULL for new diagram")
		}

		// Update the diagram
		_, err = conn.Exec(ctx, `
			UPDATE diagrams 
			SET latest_update_by = $1, latest_update_at = $2 
			WHERE id = $3
		`, userID, time.Now(), diagramID)
		if err != nil {
			t.Fatalf("Failed to update diagram: %v", err)
		}

		// Verify update
		err = conn.QueryRow(ctx, `
			SELECT latest_update_by FROM diagrams WHERE id = $1
		`, diagramID).Scan(&latestUpdateBy)
		if err != nil {
			t.Fatalf("Failed to query updated diagram: %v", err)
		}
		if latestUpdateBy == nil || *latestUpdateBy != userID {
			t.Errorf("latest_update_by should be %s, got %v", userID, latestUpdateBy)
		}
	})
}

// TestLoginColumnNames_Integration verifies the column name fix for users table
func TestLoginColumnNames_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	conn, err := PsqlConnect()
	if err != nil {
		t.Skipf("Skipping integration test - cannot connect to PostgreSQL: %v", err)
	}
	defer conn.Close(context.Background())

	ctx := context.Background()

	testUserID := uuid.NewString()
	testEmail := fmt.Sprintf("test-%s@example.com", testUserID)

	// Cleanup
	defer conn.Exec(ctx, "DELETE FROM users WHERE id = $1", testUserID)

	t.Run("Insert and query with account_created_on", func(t *testing.T) {
		// This is the fixed query that login now uses
		_, err := conn.Exec(ctx, `
			INSERT INTO users (id, email, full_name, picture_url)
			VALUES ($1, $2, $3, $4)
		`, testUserID, testEmail, "Test User", "https://example.com/pic.jpg")
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}

		// Query using the fixed column name
		var id, email, fullName, pictureUrl string
		var accountCreatedOn time.Time
		err = conn.QueryRow(ctx, `
			SELECT id, email, full_name, picture_url, account_created_on
			FROM users
			WHERE email = $1
		`, testEmail).Scan(&id, &email, &fullName, &pictureUrl, &accountCreatedOn)

		if err != nil {
			t.Fatalf("Failed to query user with account_created_on: %v", err)
		}

		if id != testUserID {
			t.Errorf("Expected user ID %s, got %s", testUserID, id)
		}

		if accountCreatedOn.IsZero() {
			t.Error("account_created_on should have a value")
		}
	})

	t.Run("Verify account_created_on has default value", func(t *testing.T) {
		var accountCreatedOn time.Time
		err := conn.QueryRow(ctx, `
			SELECT account_created_on FROM users WHERE id = $1
		`, testUserID).Scan(&accountCreatedOn)

		if err != nil {
			t.Fatalf("Failed to query account_created_on: %v", err)
		}

		// Should be within last few seconds (default CURRENT_TIMESTAMP)
		if time.Since(accountCreatedOn) > 10*time.Second {
			t.Error("account_created_on should have default CURRENT_TIMESTAMP value")
		}
	})
}
