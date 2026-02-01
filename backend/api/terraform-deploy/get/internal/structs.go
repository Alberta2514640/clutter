// Package internal provides internal types for the terraform-deploy get Lambda.
package internal

import "time"

// DeploymentLogResponse represents the response for a deployment log query.
type DeploymentLogResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	UserID      string     `json:"userId"`
	Status      string     `json:"status"`
	Log         *string    `json:"log,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
