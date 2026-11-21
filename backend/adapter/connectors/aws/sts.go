package aws

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Credential field names of an AWS connection.
const (
	CredRoleARN    = "role_arn"
	CredExternalID = "external_id"
)

var (
	// roleARNPattern is an IAM role in the commercial partition, the only one
	// the manifest's regions are in.
	roleARNPattern = regexp.MustCompile(`^arn:aws:iam::[0-9]{12}:role/[A-Za-z0-9+=,.@_/-]{1,512}$`)
	// externalIDPattern is STS's alphabet for an external id; validExternalID
	// adds its length, 2 to 1224 (over Go's repeat limit of 1000).
	externalIDPattern = regexp.MustCompile(`^[A-Za-z0-9+=,.@:/_-]+$`)
	sessionPattern    = regexp.MustCompile(`^[A-Za-z0-9+=,.@_-]{2,64}$`)
)

func validExternalID(s string) bool {
	return len(s) >= 2 && len(s) <= 1224 && externalIDPattern.MatchString(s)
}

// AssumeInput is one request for a role's short-lived keys.
type AssumeInput struct {
	RoleARN     string
	ExternalID  string
	SessionName string
	Duration    time.Duration
}

// SessionKeys are a role's short-lived keys. They are secret: they go into a
// run's environment and nowhere else.
type SessionKeys struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expires         time.Time
}

// String hides the keys from a stray log line.
func (SessionKeys) String() string { return connectors.Redacted }

// GoString hides the keys from %#v.
func (SessionKeys) GoString() string { return connectors.Redacted }

func (k SessionKeys) values() []string {
	return []string{k.AccessKeyID, k.SecretAccessKey, k.SessionToken}
}

// Assumer gets a role's short-lived keys. STSAssumer is the real one; tests
// pass a fake, and a laptop passes whatever answers for STS there.
type Assumer interface {
	AssumeRole(ctx context.Context, in AssumeInput) (SessionKeys, error)
}

// STSAPI is the part of *sts.Client STSAssumer calls.
type STSAPI interface {
	AssumeRole(ctx context.Context, in *sts.AssumeRoleInput, opts ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// STSAssumer asks AWS STS. Its client runs as B's runner role, the second
// principal the customer role's trust policy names.
type STSAssumer struct{ Client STSAPI }

// AssumeRole implements Assumer. Its errors may hold AWS's text, so they are
// for B's log only; the connector shows the user its own message.
func (a STSAssumer) AssumeRole(ctx context.Context, in AssumeInput) (SessionKeys, error) {
	if a.Client == nil {
		return SessionKeys{}, errors.New("no STS client")
	}
	out, err := a.Client.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         awssdk.String(in.RoleARN),
		RoleSessionName: awssdk.String(in.SessionName),
		ExternalId:      awssdk.String(in.ExternalID),
		DurationSeconds: awssdk.Int32(int32(in.Duration / time.Second)),
	})
	if err != nil {
		return SessionKeys{}, err
	}
	c := out.Credentials
	if c == nil || c.AccessKeyId == nil || c.SecretAccessKey == nil || c.SessionToken == nil {
		return SessionKeys{}, errors.New("STS returned no credentials")
	}
	k := SessionKeys{AccessKeyID: *c.AccessKeyId, SecretAccessKey: *c.SecretAccessKey, SessionToken: *c.SessionToken}
	if c.Expiration != nil {
		k.Expires = *c.Expiration
	}
	return k, nil
}
