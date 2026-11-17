package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/api/generic"
)

// TestGeneratedTerraformMatchesGolden pins the exact Terraform A renders for a
// set of fixture diagrams, read from the repo's template files. A refactor of
// the template code (such as moving it to generic/tftemplate for B) must leave
// these files byte-identical. A deliberate output change regenerates them with
// CLUTTER_UPDATE_GOLDEN=1 and shows up in the diff of testdata/golden.
func TestGeneratedTerraformMatchesGolden(t *testing.T) {
	fixtures := map[string]generic.DiagramLayout{
		"all-five-cors-on":  allFiveDiagram(true),
		"all-five-cors-off": allFiveDiagram(false),
		"defaults": {Nodes: []generic.DiagramNode{
			node("a", "API Gateway", map[string]any{"resource_name": "api"}),
			node("b", "Lambda", map[string]any{"resource_name": "fn"}),
			node("c", "DynamoDB", map[string]any{"resource_name": "table"}),
			node("d", "S3", map[string]any{"resource_name": "bucket"}),
			node("e", "EC2", map[string]any{"resource_name": "vm"}),
		}},
		"hostile-free-text": {Nodes: []generic.DiagramNode{
			node("a", "API Gateway", map[string]any{"resource_name": "api", "description": hostile, "stage_name": hostile}),
			node("b", "Lambda", map[string]any{"resource_name": "fn", "handler": hostile, "environment_variables": map[string]any{"V": hostile}}),
			node("c", "DynamoDB", map[string]any{"resource_name": "table", "hash_key": hostile, "enable_ttl": true, "ttl_attribute": hostile}),
			node("e", "EC2", map[string]any{"resource_name": "vm", "key_name": hostile}),
		}},
	}
	update := os.Getenv("CLUTTER_UPDATE_GOLDEN") != ""
	for name, layout := range fixtures {
		t.Run(name, func(t *testing.T) {
			tf := mustGenerate(t, layout)
			got := strings.Join([]string{
				"# resources\n" + tf.ResourcesTF,
				"# iam\n" + tf.IAMTF,
				"# outputs\n" + tf.OutputsTF,
			}, "\n")
			path := filepath.Join("testdata", "golden", name+".tf.golden")
			if update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with CLUTTER_UPDATE_GOLDEN=1 to create it)", err)
			}
			if got != string(want) {
				t.Fatalf("rendered Terraform differs from %s; if the change is intended, rerun with CLUTTER_UPDATE_GOLDEN=1 and review the diff", path)
			}
		})
	}
}
