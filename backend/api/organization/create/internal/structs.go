package internal

// Organization SQL insert structure
type OrganizationData struct {
	Id          string `json:"id"`
	CreatedBy   string `json:"created_by"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
