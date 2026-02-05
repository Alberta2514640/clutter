package resources

// getString extracts a string from variables map with a default value
func getString(vars map[string]interface{}, key, defaultVal string) string {
	if val, ok := vars[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultVal
}

// getInt extracts an int from variables map with a default value
func getInt(vars map[string]interface{}, key string, defaultVal int) int {
	if val, ok := vars[key]; ok {
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

// getBool extracts a bool from variables map with a default value
func getBool(vars map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := vars[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}
