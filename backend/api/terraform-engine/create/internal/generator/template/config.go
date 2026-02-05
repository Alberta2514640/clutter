package template


// Template paths in S3
const (
	TemplatePrefix = "templates/aws"
)

// Lambda default values
var LambdaDefaults = map[string]interface{}{
	"Handler":      "main",
	"Timeout":      3,
	"Runtime":      "provided.al2",
	"Architecture": "arm64",
	"MemorySize":   128,
}

// S3 default values
var S3Defaults = map[string]interface{}{
	"EnableVersioning":  false,
	"BlockPublicAccess": true,
}

// DynamoDB default values
var DynamoDBDefaults = map[string]interface{}{
	"HashKey":     "id",
	"HashKeyType": "S",
	"BillingMode": "PAY_PER_REQUEST",
}


// GetTemplatePath returns the S3 path for a resource template
func GetTemplatePath(resourceType string, templateFile string) string {
	return TemplatePrefix + "/" + resourceType + "/" + templateFile
}
