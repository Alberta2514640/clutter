package internal

type RequestBody struct {
	OrganizationId string `json:"organizationId"`
	ProjectId      string `json:"projectId"`
	DiagramId      string `json:"diagramId"`
	Command        string `json:"command"`
}
