package aws

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/connectors/contract"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/google/uuid"
)

// TestContract runs the shared connector contract with a fake STS and a fake
// terraform behind the real job runner and Terraform runner.
func TestContract(t *testing.T) {
	contract.Run(t, contract.Fixture{
		New: func() connectors.Connector {
			c, _, _ := newTestConnector(t)
			return c
		},
		Region:   "ca-central-1",
		Good:     connectors.Secrets{CredRoleARN: goodRole, CredExternalID: goodExternal},
		Bad:      connectors.Secrets{CredRoleARN: "arn:aws:iam::210987654321:role/NotClutters", CredExternalID: "not-the-external-id"},
		Graph:    fullGraph(),
		Settings: map[string]any{"function_runtime": "python3.12", "instance_type": "t3.large"},
		Cross: &contract.CrossFixture{
			Graph: connectors.Graph{
				Blocks: map[string]connectors.Block{
					"src": {ID: "src", Kind: connectors.KindVM, Name: "web", Fields: map[string]any{}},
					"dst": {ID: "dst", Kind: connectors.KindVM, Name: "db", Fields: map[string]any{"public": true}},
				},
				Edges: []connectors.Edge{{Source: "src", Target: "dst"}},
			},
			Placed:   []string{"dst"},
			SourceVM: "src",
			TargetVM: "dst",
		},
	})
}

// TestRunKeysLiveOnlyInTheEnvironment checks that a run gets the session keys
// and region in its environment, never in a file, and that a key Terraform
// prints is removed from the run's log.
func TestRunKeysLiveOnlyInTheEnvironment(t *testing.T) {
	c, fs, ft := newTestConnector(t)
	d := testDeployment("ca-central-1")
	var log bytes.Buffer
	runID := uuid.NewString()
	res, err := c.Plan(context.Background(), connectors.Run{ID: runID, Action: connectors.ActionPlan, Log: &log}, fullGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	if res.Add == 0 {
		t.Fatal("the plan summary was not read")
	}
	if ft.leaked {
		t.Fatal("a session key was written into the run's folder")
	}
	for _, env := range ft.envs {
		want := map[string]string{
			"AWS_ACCESS_KEY_ID": fakeKeyID, "AWS_SECRET_ACCESS_KEY": fakeSecret, "AWS_SESSION_TOKEN": fakeToken,
			"AWS_REGION": "ca-central-1", "AWS_EC2_METADATA_DISABLED": "true",
			// Every role and user the run creates names the stack's
			// permissions boundary for its kind.
			boundaryEnv:     "arn:aws:iam::123456789012:policy/clutter/ClutterPermissionsBoundary-ab12cd34",
			userBoundaryEnv: "arn:aws:iam::123456789012:policy/clutter/ClutterUserBoundary-ab12cd34",
		}
		for k, v := range want {
			if env[k] != v {
				t.Errorf("run environment %s = %q, want %q", k, env[k], v)
			}
		}
		for k := range env {
			if strings.HasPrefix(k, "TF_LOG") || k == "AWS_PROFILE" || k == "AWS_SHARED_CREDENTIALS_FILE" {
				t.Errorf("run environment holds %s", k)
			}
		}
	}
	for _, v := range []string{fakeKeyID, fakeSecret, fakeToken} {
		if strings.Contains(log.String(), v) {
			t.Fatal("the run's log holds a session key")
		}
	}
	if !strings.Contains(log.String(), "debug: using key [redacted]") {
		t.Fatalf("the printed key was not redacted: %s", log.String())
	}

	last := fs.calls[len(fs.calls)-1]
	if last.Duration != runDuration || last.SessionName != "clutter-adapter-"+strings.ReplaceAll(runID, "-", "") {
		t.Errorf("run session %q for %s", last.SessionName, last.Duration)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9+=,.@_-]{2,64}$`).MatchString(last.SessionName) {
		t.Errorf("session name %q is not one STS takes", last.SessionName)
	}
}

func TestApplyReportsWhatItBuilt(t *testing.T) {
	c, _, _ := newTestConnector(t)
	d := testDeployment("ca-central-1")
	res, err := c.Apply(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionApply, Log: &bytes.Buffer{}}, fullGraph(), d)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"web", "net1", "fw", "dndnode_3", "bucket", "fn", "table", "api"} {
		if !strings.HasPrefix(res.Resources[id], "fake-") {
			t.Errorf("node %s: built %q", id, res.Resources[id])
		}
	}
}

func TestDestroyNeedsOnlyTheProvider(t *testing.T) {
	c, _, ft := newTestConnector(t)
	d := testDeployment("ca-central-1")
	if _, err := c.Destroy(context.Background(), connectors.Run{ID: uuid.NewString(), Action: connectors.ActionDestroy, Log: &bytes.Buffer{}}, d); err != nil {
		t.Fatal(err)
	}
	if got := ft.args[len(ft.args)-1][0]; got != "destroy" {
		t.Fatalf("last step %q", got)
	}
}

func TestVerifyShowsNoAWSText(t *testing.T) {
	c, fs, _ := newTestConnector(t)
	conn := testDeployment("ca-central-1").Connection
	if err := c.VerifyConnection(context.Background(), conn); err != nil {
		t.Fatalf("good role refused: %v", err)
	}
	if got := fs.calls[0]; got.Duration != verifyDuration || got.RoleARN != goodRole || got.ExternalID != goodExternal {
		t.Fatalf("verify asked STS for %+v", got)
	}

	// A stack role STS refuses (another account's) reaches STS.
	conn.Credentials = connectors.Secrets{CredRoleARN: "arn:aws:iam::210987654321:role/AllowClutterToDeployTerraformRole-ef56ab78", CredExternalID: goodExternal}
	calls := len(fs.calls)
	err := c.VerifyConnection(context.Background(), conn)
	if connectors.UserMessage(err, "") == "" || strings.Contains(err.Error(), "AccessDenied") || len(fs.calls) != calls+1 {
		t.Fatalf("a refusal should be a user error without AWS's text: %v", err)
	}

	before := len(fs.calls)
	for _, bad := range []connectors.Secrets{
		// A well-formed role the Clutter stack did not create has no boundary.
		{CredRoleARN: "arn:aws:iam::123456789012:role/SomeOtherRole", CredExternalID: goodExternal},
		{CredRoleARN: "arn:aws:iam::12345:role/x", CredExternalID: goodExternal},
		{CredRoleARN: goodRole + `"`, CredExternalID: goodExternal},
		{CredRoleARN: goodRole, CredExternalID: "x"},
		{CredRoleARN: goodRole, CredExternalID: "has space"},
	} {
		conn.Credentials = bad
		if err := c.VerifyConnection(context.Background(), conn); connectors.UserMessage(err, "") == "" {
			t.Errorf("malformed credentials not refused as a user error: %v", err)
		}
	}
	if len(fs.calls) != before {
		t.Error("malformed credentials reached STS")
	}
}

type fakeSTSAPI struct {
	in  *sts.AssumeRoleInput
	out *sts.AssumeRoleOutput
	err error
}

func (f *fakeSTSAPI) AssumeRole(_ context.Context, in *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.in = in
	return f.out, f.err
}

func TestSTSAssumerSendsTheExternalID(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	api := &fakeSTSAPI{out: &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
		AccessKeyId: ptr(fakeKeyID), SecretAccessKey: ptr(fakeSecret), SessionToken: ptr(fakeToken), Expiration: &exp,
	}}}
	keys, err := STSAssumer{Client: api}.AssumeRole(context.Background(), AssumeInput{RoleARN: goodRole, ExternalID: goodExternal, SessionName: "clutter-adapter-x", Duration: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if *api.in.ExternalId != goodExternal || *api.in.RoleArn != goodRole || *api.in.DurationSeconds != 3600 {
		t.Fatalf("sent %+v", api.in)
	}
	if keys.SessionToken != fakeToken || !keys.Expires.Equal(exp) {
		t.Fatal("keys not read")
	}
	if s := keys.String(); strings.Contains(s, fakeSecret) {
		t.Fatal("keys print their values")
	}

	api.out, api.err = nil, errors.New("AccessDenied")
	if _, err := (STSAssumer{Client: api}).AssumeRole(context.Background(), AssumeInput{}); err == nil {
		t.Fatal("an STS error was swallowed")
	}
	api.out, api.err = &sts.AssumeRoleOutput{}, nil
	if _, err := (STSAssumer{Client: api}).AssumeRole(context.Background(), AssumeInput{}); err == nil {
		t.Fatal("an answer without keys was accepted")
	}
}

func ptr(s string) *string { return &s }

// TestRoleTemplateMatches keeps the CloudFormation role template and this
// connector in step: the version output equals RoleTemplateVersion, B's
// runner role is a trusted principal, and the external id condition stays.
func TestRoleTemplateMatches(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(backendDir(), "infra", "modules", "s3", "uploads", "client_side_terraform_deployer_role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	m := regexp.MustCompile(`(?m)^  RoleTemplateVersion:\n    Value: "([^"]+)"$`).FindStringSubmatch(body)
	if m == nil || m[1] != RoleTemplateVersion {
		t.Fatalf("the role template's RoleTemplateVersion output is %v, the connector expects %q", m, RoleTemplateVersion)
	}
	for _, want := range []string{
		`role/${AdapterRunnerRoleName}`,
		`role/TerraformDeployerFargateTaskRole`,
		`sts:ExternalId: !Ref ExternalId`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the role template lacks %q", want)
		}
	}
}
