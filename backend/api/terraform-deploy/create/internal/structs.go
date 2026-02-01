// Package internal provides internal types for the terraform-deploy create Lambda.
package internal

import "time"

// DeployRequest represents the incoming request body for triggering a deployment.
type DeployRequest struct {
	ProjectID string `json:"projectId"`
}

// DeploymentRun represents a deployment run record in the database.
type DeploymentRun struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	UserID      string     `json:"userId"`
	Status      string     `json:"status"`
	Log         *string    `json:"log,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ECSConfig holds the ECS task configuration passed via environment variables.
type ECSConfig struct {
	ClusterARN        string
	TaskDefinitionARN string
	SubnetIDs         []string
	SecurityGroupID   string
	S3Bucket          string
	AWSRegion         string
}

// DeployResponse represents the response returned after triggering a deployment.
type DeployResponse struct {
	RunID     string `json:"runId"`
	ProjectID string `json:"projectId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}
