package template

import (
	"bytes"
	"fmt"
	"text/template"
)

type RenderContext struct {
	ResourceName string
	Variables    map[string]interface{}
}

// Render executes a template with the given variables
func Render(tmpl *Template, vars map[string]interface{}) (string, error) {
	t, err := template.New(tmpl.Name).Parse(tmpl.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", tmpl.Name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", tmpl.Name, err)
	}

	return buf.String(), nil
}

// RenderWithDefaults renders a template, filling missing variables with defaults
func RenderWithDefaults(tmpl *Template, vars map[string]interface{}, defaults map[string]interface{}) (string, error) {
	merged := make(map[string]interface{})

	// Apply defaults first
	for k, v := range defaults {
		merged[k] = v
	}

	// Override with provided variables
	for k, v := range vars {
		merged[k] = v
	}

	return Render(tmpl, merged)
}
