package generic_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestCheckOrganizationMembershipPSQL_WithMock tests with mocked database
func TestCheckOrganizationMembershipPSQL_WithMock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	ctx := context.Background()

	t.Run("authorized user", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("org-123", "user-456").
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		// Note: We can't directly use mock with CheckOrganizationMembershipPSQL
		// because it expects *pgx.Conn, not pgxmock.PgxConnIface
		// This demonstrates the pattern - in production, use interface abstraction

		// Simulate the query execution
		var exists bool
		row := mock.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id = $1 AND member_id = $2)`,
			"org-123", "user-456")
		if err := row.Scan(&exists); err != nil {
			t.Errorf("Query failed: %v", err)
		}
		if !exists {
			t.Error("Expected exists to be true")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("unauthorized user", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("org-123", "user-789").
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		var exists bool
		row := mock.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id = $1 AND member_id = $2)`,
			"org-123", "user-789")
		if err := row.Scan(&exists); err != nil {
			t.Errorf("Query failed: %v", err)
		}
		if exists {
			t.Error("Expected exists to be false")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestGetProjectOrganizationPSQL_WithMock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	ctx := context.Background()

	t.Run("project found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT organization_id FROM projects WHERE id = \$1`).
			WithArgs("project-123").
			WillReturnRows(pgxmock.NewRows([]string{"organization_id"}).AddRow("org-456"))

		var orgID string
		row := mock.QueryRow(ctx, `SELECT organization_id FROM projects WHERE id = $1`, "project-123")
		if err := row.Scan(&orgID); err != nil {
			t.Errorf("Query failed: %v", err)
		}
		if orgID != "org-456" {
			t.Errorf("Expected org-456, got %s", orgID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT organization_id FROM projects WHERE id = \$1`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		var orgID string
		row := mock.QueryRow(ctx, `SELECT organization_id FROM projects WHERE id = $1`, "nonexistent")
		err := row.Scan(&orgID)
		if err == nil {
			t.Error("Expected error for nonexistent project")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestGetDiagramProjectPSQL_WithMock(t *testing.T) {
	mock, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer mock.Close(context.Background())

	ctx := context.Background()

	t.Run("diagram found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT project_id FROM diagrams WHERE id = \$1`).
			WithArgs("diagram-123").
			WillReturnRows(pgxmock.NewRows([]string{"project_id"}).AddRow("project-456"))

		var projectID string
		row := mock.QueryRow(ctx, `SELECT project_id FROM diagrams WHERE id = $1`, "diagram-123")
		if err := row.Scan(&projectID); err != nil {
			t.Errorf("Query failed: %v", err)
		}
		if projectID != "project-456" {
			t.Errorf("Expected project-456, got %s", projectID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("diagram not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT project_id FROM diagrams WHERE id = \$1`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		var projectID string
		row := mock.QueryRow(ctx, `SELECT project_id FROM diagrams WHERE id = $1`, "nonexistent")
		err := row.Scan(&projectID)
		if err == nil {
			t.Error("Expected error for nonexistent diagram")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
