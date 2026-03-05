package generic

// AllowedJobStatuses defines valid job status values for the Ansible Fargate system
var AllowedJobStatuses = map[string]bool{
	"QUEUED":    true,
	"STARTING":  true,
	"RUNNING":   true,
	"COMPLETED": true,
	"FAILED":    true,
}
