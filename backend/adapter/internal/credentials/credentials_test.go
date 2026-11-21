package credentials

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/google/uuid"
)

func newKey(t *testing.T) []byte {
	k := make([]byte, KeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatal(err)
	}
	return k
}

const secretValue = "app-cred-secret-7f3a9c"

func TestLocalRoundTripIsEncrypted(t *testing.T) {
	dir := t.TempDir()
	key := newKey(t)
	s, err := NewLocal(dir, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id := uuid.NewString()
	ref, err := s.Put(ctx, id, connectors.Secrets{"secret": secretValue, "id": "cred-id"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ref, secretValue) {
		t.Fatal("the reference holds the secret")
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".secret"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secretValue)) || bytes.Contains(raw, []byte("cred-id")) {
		t.Fatal("the file holds the secret in clear")
	}
	got, err := s.Get(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got["secret"] != secretValue || got["id"] != "cred-id" {
		t.Fatal("round trip lost a value")
	}

	other, _ := NewLocal(dir, newKey(t))
	if _, err := other.Get(ctx, ref); err == nil {
		t.Fatal("decrypted with another key")
	}

	// A file copied to another reference does not decrypt.
	id2 := uuid.NewString()
	if err := os.WriteFile(filepath.Join(dir, id2+".secret"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, localPrefix+id2); err == nil {
		t.Fatal("a moved secret decrypted")
	}

	if err := s.Delete(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if err := s.Delete(ctx, ref); err != nil {
		t.Fatalf("second delete: %v", err)
	}
}

func TestLocalRefusesBadReferences(t *testing.T) {
	s, err := NewLocal(t.TempDir(), newKey(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"", "local:../x", "local:not-a-uuid", "arn:aws:secretsmanager:x", "local:" + strings.ToUpper(uuid.NewString())} {
		if _, err := s.Get(context.Background(), ref); err == nil || errors.Is(err, ErrNotFound) {
			t.Fatalf("reference %q: %v", ref, err)
		}
	}
	if _, err := s.Put(context.Background(), "../escape", connectors.Secrets{}); err == nil {
		t.Fatal("a bad connection id was accepted")
	}
}

func TestParseKeyErrorsHideTheKey(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("short-key-material"))
	for _, in := range []string{"not base64 !!", short} {
		_, err := ParseKey(in)
		if err == nil {
			t.Fatalf("%q accepted", in)
		}
		if strings.Contains(err.Error(), in) || strings.Contains(err.Error(), "short-key-material") {
			t.Fatal("the error repeats the key")
		}
	}
	good := base64.StdEncoding.EncodeToString(newKey(t))
	if _, err := ParseKey(good); err != nil {
		t.Fatal(err)
	}
}

func TestSecretsAreRedactedWhenPrinted(t *testing.T) {
	s := connectors.Secrets{"secret": secretValue}
	for _, out := range []string{fmt.Sprint(s), fmt.Sprintf("%v %+v %#v %s", s, s, s, s)} {
		if strings.Contains(out, secretValue) {
			t.Fatalf("printed: %s", out)
		}
	}
	type wrap struct{ Credentials connectors.Secrets }
	if strings.Contains(fmt.Sprintf("%+v", wrap{s}), secretValue) {
		t.Fatal("printed inside a struct")
	}
}

type fakeSM struct {
	secrets map[string]string
}

func (f *fakeSM) CreateSecret(_ context.Context, in *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	arn := "arn:aws:secretsmanager:ca-central-1:000000000000:secret:" + *in.Name
	f.secrets[arn] = *in.SecretString
	return &secretsmanager.CreateSecretOutput{ARN: aws.String(arn)}, nil
}

func (f *fakeSM) GetSecretValue(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	v, ok := f.secrets[*in.SecretId]
	if !ok {
		return nil, &smtypes.ResourceNotFoundException{}
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(v)}, nil
}

func (f *fakeSM) DeleteSecret(_ context.Context, in *secretsmanager.DeleteSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	if _, ok := f.secrets[*in.SecretId]; !ok {
		return nil, &smtypes.ResourceNotFoundException{}
	}
	if in.RecoveryWindowInDays == nil || *in.RecoveryWindowInDays != RecoveryWindowDays {
		return nil, errors.New("unexpected recovery window")
	}
	delete(f.secrets, *in.SecretId)
	return &secretsmanager.DeleteSecretOutput{}, nil
}

func TestSecretsManagerRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := SecretsManager{Client: &fakeSM{secrets: map[string]string{}}, Prefix: "clutter/b/connections/"}
	id := uuid.NewString()
	ref, err := s.Put(ctx, id, connectors.Secrets{"secret": secretValue})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(ref, "clutter/b/connections/"+id) || strings.Contains(ref, secretValue) {
		t.Fatalf("reference %q", ref)
	}
	got, err := s.Get(ctx, ref)
	if err != nil || got["secret"] != secretValue {
		t.Fatalf("get: %v", err)
	}
	if err := s.Delete(ctx, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if err := s.Delete(ctx, ref); err != nil {
		t.Fatalf("second delete: %v", err)
	}
	if _, err := s.Get(ctx, "local:"+id); err == nil {
		t.Fatal("a local reference was accepted")
	}
}
