package main

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestAnsibleTaskGetsOnlyThisJobsRoleAndNoDatabase checks the overrides the
// deployed handler starts the runner with: this job's role and external id,
// and nothing that names or holds a database. The runner reports its result
// in a status file instead (generic.ReconcileJob).
func TestAnsibleTaskGetsOnlyThisJobsRoleAndNoDatabase(t *testing.T) {
	t.Setenv("PSQL_CONNECTION_STRING", "postgres://clutter:secret@db.example:5432/postgres")
	msg := map[string]string{
		"job_id":                  "66666666-6666-4666-8666-666666666666",
		"playbook_s3_key":         "33333333-3333-4333-8333-333333333333/11111111-1111-4111-8111-111111111111/22222222-2222-4222-8222-222222222222/playbooks/a-site.yml",
		"target_instance_ids":     `["i-0123456789abcdef0"]`,
		"extra_vars":              "{}",
		"org_id":                  "33333333-3333-4333-8333-333333333333",
		"project_id":              "11111111-1111-4111-8111-111111111111",
		"diagram_id":              "22222222-2222-4222-8222-222222222222",
		"role_arn":                "arn:aws:iam::111122223333:role/AllowClutterToDeployTerraformRole-abc",
		"assume_role_external_id": "this-orgs-external-id",
	}
	in := buildAnsibleRunTaskInput("cluster", "task-def", []string{"subnet-1"}, "sg-1", "clutter-bucket", msg)

	if in.Overrides == nil || len(in.Overrides.ContainerOverrides) != 1 {
		t.Fatalf("want one container override, got %+v", in.Overrides)
	}
	override := in.Overrides.ContainerOverrides[0]
	if len(override.EnvironmentFiles) != 0 {
		t.Fatalf("the task gets environment files: %+v", override.EnvironmentFiles)
	}
	want := map[string]string{
		"JOB_ID":                  msg["job_id"],
		"PLAYBOOK_S3_KEY":         msg["playbook_s3_key"],
		"TARGET_INSTANCE_IDS":     "i-0123456789abcdef0",
		"EXTRA_VARS":              "{}",
		"ORG_ID":                  msg["org_id"],
		"PROJECT_ID":              msg["project_id"],
		"DIAGRAM_ID":              msg["diagram_id"],
		"CLIENT_ROLE_ARN":         msg["role_arn"],
		"ASSUME_ROLE_EXTERNAL_ID": msg["assume_role_external_id"],
		"S3_BUCKET_NAME":          "clutter-bucket",
	}
	got := map[string]string{}
	for _, pair := range override.Environment {
		name, value := aws.ToString(pair.Name), aws.ToString(pair.Value)
		if _, dup := got[name]; dup {
			t.Fatalf("%s is set twice", name)
		}
		got[name] = value
		lower := strings.ToLower(name + "=" + value)
		for _, word := range []string{"psql", "postgres", "database", "secret"} {
			if strings.Contains(lower, word) {
				t.Errorf("override %s holds %q", name, word)
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("overrides %v, want %v", got, want)
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
}
