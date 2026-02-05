package internal

import (
	"github.com/Alberta2514640/clutter/backend/api/generic"
)

type SupabaseWebhookPayload struct {
	Type      string               `json:"type"`                 // INSERT, UPDATE, DELETE
	Table     string               `json:"table"`                // table name
	Schema    string               `json:"schema"`               // usually "public"
	Record    generic.DiagramRecord `json:"record"`              // the new/updated record
	OldRecord generic.DiagramRecord `json:"old_record,omitempty"` // for UPDATE
}

// ResourceType represents supported AWS resource types
type ResourceType string

const (
	ResourceTypeLambda     ResourceType = "Lambda"
	ResourceTypeDynamoDB   ResourceType = "DynamoDB"
	ResourceTypeS3         ResourceType = "S3"
	ResourceTypeAPIGateway ResourceType = "API Gateway"
)

// TerraformResource represents a parsed diagram node ready for TF generation
type TerraformResource struct {
	ID           string
	Name         string                 // Sanitized name for TF resource naming
	Type         ResourceType
	Variables    map[string]interface{}
	SourceNodeID string // Original node ID for edge mapping
}

// IAMRelationship represents a connection requiring IAM permissions
type IAMRelationship struct {
	SourceResource *TerraformResource
	TargetResource *TerraformResource
	EdgeID         string
	EdgeLabel      string
}

// GeneratedTerraform holds all generated TF content
type GeneratedTerraform struct {
	OrgID       string
	ProjectID   string
	DiagramID   string
	MainTF      string
	ResourcesTF string
	IAMTF       string
	VariablesTF string
	OutputsTF   string
}

// GenerationError provides detailed error information
type GenerationError struct {
	NodeID    string `json:"node_id"`
	NodeLabel string `json:"node_label"`
	ErrorType string `json:"error_type"` // "unknown_label", "missing_variable", "invalid_config"
	Message   string `json:"message"`
}

// ResourceGenerator interface for each resource type
type ResourceGenerator interface {
	Generate(node generic.DiagramNode, resourceName string) (string, error)
	ValidateVariables(variables map[string]interface{}) error
	GetRequiredVariables() []string
	GetOutputs(resourceName string) string
}
