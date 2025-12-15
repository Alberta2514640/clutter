package internal

// Incoming request body structure
type RequestBody struct {
	OrganizationName string `json:"organizationName"`
	Description      string `json:"description"`
}

// Organization SQL insert structure
type OrganizationData struct {
	Id          string `json:"id"`
	CreatedBy   string `json:"created_by"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
