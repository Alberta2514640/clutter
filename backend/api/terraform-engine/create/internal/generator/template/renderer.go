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