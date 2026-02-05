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

// GetString extracts a string from variables with a default fallback
func GetString(vars map[string]interface{}, key string, defaultVal string) string {
	if val, exists := vars[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultVal
}

// GetInt extracts an int from variables with a default fallback
func GetInt(vars map[string]interface{}, key string, defaultVal int) int {
	if val, exists := vars[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return defaultVal
}

// GetBool extracts a bool from variables with a default fallback
func GetBool(vars map[string]interface{}, key string, defaultVal bool) bool {
	if val, exists := vars[key]; exists {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultVal
}

// GetStringMap extracts a map[string]string from variables with a default fallback
func GetStringMap(vars map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if val, exists := vars[key]; exists {
		if mapVal, ok := val.(map[string]interface{}); ok {
			for k, v := range mapVal {
				if strVal, ok := v.(string); ok {
					result[k] = strVal
				}
			}
		}
	}
	return result
}

// GetTemplatePath returns the S3 path for a resource template
func GetTemplatePath(resourceType string, templateFile string) string {
	return TemplatePrefix + "/" + resourceType + "/" + templateFile
}
