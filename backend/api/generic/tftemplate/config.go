package tftemplate

// Template paths in S3
const (
	TemplatePrefix = "templates/aws"
)

// GetTemplatePath returns the S3 path for a resource template
func GetTemplatePath(resourceType string, templateFile string) string {
	return TemplatePrefix + "/" + resourceType + "/" + templateFile
}
