package generic

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

// CheckOrganizationMembershipPSQL verifies if a user is a member of an organization using PostgreSQL
// Returns nil if authorized, error otherwise
func CheckOrganizationMembershipPSQL(
	ctx context.Context,
	conn Querier,
	userID string,
	orgID string,
) error {
	if userID == "" || orgID == "" {
		return NewForbiddenError("user does not have access to this organization")
	}

	// Check if org exists
	queryOrgExists := `
		SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)
	`
	var orgExists bool
	if err := conn.QueryRow(ctx, queryOrgExists, orgID).Scan(&orgExists); err != nil {
		return err
	}

	// Neither message names an id: a response must not confirm which ids exist.
	if !orgExists {
		return NewNotFoundError("organization not found")
	}

	var userIsMemberOfOrg bool
	queryUserIsMember := `
		SELECT EXISTS(
			SELECT 1 
			FROM organization_members 
			WHERE organization_id = $1 AND member_id = $2
		)
	`
	err := conn.QueryRow(ctx, queryUserIsMember, orgID, userID).Scan(&userIsMemberOfOrg)
	if err != nil {
		return err
	}

	if !userIsMemberOfOrg {
		return NewForbiddenError("user does not have access to this organization")
	}

	return nil
}

// CheckOwnershipPSQL proves the caller may act on every id one request names
// and returns the organization's id in canonical form. CheckOwnedIDsPSQL says
// what it checks; use that one when a key or a path is built from the ids.
func CheckOwnershipPSQL(
	ctx context.Context,
	conn Querier,
	userID string,
	orgID string,
	projectID string,
	diagramID string,
) (string, error) {
	ids, err := CheckOwnedIDsPSQL(ctx, conn, userID, orgID, projectID, diagramID)
	return ids.OrgID, err
}

// OwnedIDs are the ids of one request after CheckOwnedIDsPSQL has checked
// them, in the canonical form Postgres returns. Build keys and paths from these,
// never from the request's text.
type OwnedIDs struct {
	OrgID     string
	ProjectID string
	DiagramID string
}

// CheckOwnedIDsPSQL proves the caller may act on every id one request names:
// the diagram belongs to the project, the project to the organization, and the
// user is a member of that organization. Membership alone is never enough.
//
// projectID and diagramID may be empty. orgID may be empty when a child id is
// given; the organization then comes from the project row, never from the
// request. ProjectID is filled from the diagram row when only a diagram is
// named.
//
// Every error is an *AuthorizationError or a database error. Every 404 carries
// the one message "not found", whether the id is not a UUID, does not exist,
// or belongs to another parent, so a caller cannot learn that another tenant's
// id exists. When the caller named the organization, a non-member gets the 403
// of CheckOrganizationMembershipPSQL before any child id is looked up; when the
// organization was found from a child id, a non-member gets the 404.
func CheckOwnedIDsPSQL(
	ctx context.Context,
	conn Querier,
	userID string,
	orgID string,
	projectID string,
	diagramID string,
) (OwnedIDs, error) {
	if userID == "" {
		return OwnedIDs{}, NewForbiddenError("user does not have access to this organization")
	}
	ids := []*string{&orgID, &projectID, &diagramID}
	for _, id := range ids {
		if *id == "" {
			continue
		}
		canonical, ok := CanonicalUuid(*id)
		if !ok || !IsValidUuid(*id) {
			return OwnedIDs{}, NewNotFoundError("not found")
		}
		*id = canonical
	}

	callerNamedOrg := orgID != ""
	if callerNamedOrg {
		err := CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID)
		var authErr *AuthorizationError
		if errors.As(err, &authErr) && authErr.StatusCode == http.StatusNotFound {
			return OwnedIDs{}, NewNotFoundError("not found")
		}
		if err != nil {
			return OwnedIDs{}, err
		}
	}

	if diagramID != "" {
		owner, err := GetDiagramProjectPSQL(ctx, conn, diagramID)
		if err != nil {
			return OwnedIDs{}, sameNotFound(err)
		}
		if projectID != "" && owner != projectID {
			return OwnedIDs{}, NewNotFoundError("not found")
		}
		projectID = owner
	}

	if projectID != "" {
		owner, err := GetProjectOrganizationPSQL(ctx, conn, projectID)
		if err != nil {
			return OwnedIDs{}, sameNotFound(err)
		}
		if orgID != "" && owner != orgID {
			return OwnedIDs{}, NewNotFoundError("not found")
		}
		orgID = owner
	}

	if !callerNamedOrg {
		if orgID == "" {
			return OwnedIDs{}, NewForbiddenError("user does not have access to this organization")
		}
		err := CheckOrganizationMembershipPSQL(ctx, conn, userID, orgID)
		var authErr *AuthorizationError
		if errors.As(err, &authErr) {
			return OwnedIDs{}, NewNotFoundError("not found")
		}
		if err != nil {
			return OwnedIDs{}, err
		}
	}

	return OwnedIDs{OrgID: orgID, ProjectID: projectID, DiagramID: diagramID}, nil
}

// sameNotFound gives every authorization error from a child lookup the one
// message CheckOwnershipPSQL uses, so a missing project reads the same as a
// project of another organization. Database errors pass through.
func sameNotFound(err error) error {
	var authErr *AuthorizationError
	if errors.As(err, &authErr) {
		return NewNotFoundError("not found")
	}
	return err
}

// CanonicalUuid returns the lowercase hyphenated form Postgres returns, so ids
// from a request compare equal to ids from a row. ok is false when id is not
// a UUID of any version.
func CanonicalUuid(id string) (string, bool) {
	u, err := uuid.Parse(id)
	if err != nil {
		return "", false
	}
	return u.String(), true
}

// AuthorizationResponse maps an error from the ownership checks to a response:
// an *AuthorizationError keeps its status and message, anything else is a
// generic 500 with the cause in the log only.
func AuthorizationResponse(err error) (events.APIGatewayProxyResponse, error) {
	var authErr *AuthorizationError
	if errors.As(err, &authErr) {
		return Response(authErr.StatusCode, Json{"message": authErr.Message})
	}
	log.Printf("ownership check failed: %v", err)
	return Response(http.StatusInternalServerError, Json{"message": "failed to check authorization"})
}
