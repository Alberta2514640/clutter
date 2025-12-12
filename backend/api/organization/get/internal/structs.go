package internal

import "time"

// Data required for organization overview
type OrgOverviewData struct {
	Id            string    `json:"id"`
	CreatedBy     string    `json:"created_by"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	TotalProjects int       `json:"total_projects"`
	TotalDiagrams int       `json:"total_diagrams"`
	TotalMembers  int       `json:"total_members"`
	Members       []string  `json:"members"`
}
