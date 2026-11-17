package tftemplate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// NewTemplateLoaderFromDir returns a loader that serves the template files under
// dir, a local copy of the provider folder that is uploaded to TemplatePrefix
// (backend/infra/modules/s3/uploads/aws in the repo). It reads every file once
// and never reaches S3, so tests can render the real templates offline. A
// template that is not in dir fails to load as one missing from the bucket does.
func NewTemplateLoaderFromDir(dir string) (*TemplateLoader, error) {
	cache := make(map[string]*Template)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		key := TemplatePrefix + "/" + filepath.ToSlash(rel)
		cache[key] = &Template{Name: key, Content: string(content)}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read templates from %s: %w", dir, err)
	}
	if len(cache) == 0 {
		return nil, fmt.Errorf("no templates under %s", dir)
	}
	return &TemplateLoader{cache: cache}, nil
}
