package internal

type RequestBody struct {
	OrganizationId      string `json:"organizationId"`
	ProjectId           string `json:"projectId"`
	DiagramId           string `json:"diagramId"`
	AccountAccessRoleId string `json:"accountAccessRoleId"`
	Region              string `json:"region"`
	Command             string `json:"command"`
}
