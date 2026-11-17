package aws

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// verifySession is the STS session name VerifyConnection uses.
const verifySession = "clutter-adapter-verify"

// ErrNeedsRealAWS is what a run gets on a laptop, where no AWS account exists.
var ErrNeedsRealAWS = &connectors.UserError{Message: "This needs a real AWS account. Locally, Clutter renders and checks the Terraform for AWS but cannot run it."}

// LocalAssumer stands in for STS on a laptop (A2). It accepts any well-formed
// role at verification, so A2's linked fake account shows as a connection, and
// refuses every run with ErrNeedsRealAWS. It never returns usable keys. main.go
// allows it only with the local credential store, so the cloud never uses it.
type LocalAssumer struct{}

// AssumeRole implements Assumer.
func (LocalAssumer) AssumeRole(_ context.Context, in AssumeInput) (SessionKeys, error) {
	if in.SessionName != verifySession {
		return SessionKeys{}, ErrNeedsRealAWS
	}
	return SessionKeys{AccessKeyID: "local", SecretAccessKey: "local", SessionToken: "local", Expires: time.Now().Add(in.Duration)}, nil
}

// localEndpoint checks an endpoint on this machine, such as LocalStack's, and
// returns it as scheme://host:port. It refuses any other host, so a laptop
// setting can never send AWS calls or keys to another machine.
func localEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.Port() == "" {
		return "", errors.New("the AWS endpoint must look like http://127.0.0.1:4566")
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return "", errors.New("the AWS endpoint must be on this machine (127.0.0.1, ::1 or localhost)")
	}
	return u.Scheme + "://" + u.Host, nil
}

// NewLocalStackAssumer asks LocalStack's STS (backend/devserver/localstack.sh)
// for a role's keys, as STSAssumer asks AWS. It signs with LocalStack's test
// keys, never this machine's AWS credentials. LocalStack 4.14.0 does not
// enforce IAM: it answers for any role ARN, whether the role exists or not,
// and ignores the external id, so a verify here proves the call path, not the
// role's trust policy. main.go allows it only with the local credential store.
func NewLocalStackAssumer(endpoint string) (STSAssumer, error) {
	ep, err := localEndpoint(endpoint)
	if err != nil {
		return STSAssumer{}, err
	}
	client := sts.New(sts.Options{
		Region:       "us-east-1",
		BaseEndpoint: awssdk.String(ep),
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
	})
	return STSAssumer{Client: client}, nil
}
