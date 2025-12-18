package internal

import "time"

// Data required for organization overview
type OrgOverviewData struct {
	Id            string    `json:"id"`
	CreatedBy     string    `json:"created_by"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	TotalMembers  int       `json:"total_members"`
	Members       []string  `json:"members"`
	TotalProjects int       `json:"total_projects"`
	Projects      []Project `json:"projects"`
	TotalDiagrams int       `json:"total_diagrams"`
}

// Data that will be sent for projects in an organization
type Project struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
