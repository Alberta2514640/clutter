package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/google/uuid"
)

// SecretsManagerAPI is the part of the Secrets Manager client the store uses.
type SecretsManagerAPI interface {
	CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

// RecoveryWindowDays is how long a deleted connection's secret can be restored.
const RecoveryWindowDays = 7

// SecretsManager keeps one secret per connection, named Prefix + connection
// id. The reference is the secret's ARN.
type SecretsManager struct {
	Client SecretsManagerAPI
	// Prefix starts every secret name, such as "clutter/b/connections/".
	Prefix string
}

// Put implements Store.
func (s SecretsManager) Put(ctx context.Context, connectionID string, secrets connectors.Secrets) (string, error) {
	if _, err := uuid.Parse(connectionID); err != nil {
		return "", errors.New("connection id is not valid")
	}
	plain, err := json.Marshal(map[string]string(secrets))
	if err != nil {
		return "", err
	}
	out, err := s.Client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(s.Prefix + connectionID),
		SecretString: aws.String(string(plain)),
		Description:  aws.String("Clutter connection credentials"),
	})
	if err != nil {
		return "", err
	}
	if out.ARN == nil || *out.ARN == "" {
		return "", errors.New("secrets manager returned no ARN")
	}
	return *out.ARN, nil
}

func checkARN(ref string) error {
	if !strings.HasPrefix(ref, "arn:") || !strings.Contains(ref, ":secretsmanager:") {
		return errors.New("not a secrets manager reference")
	}
	return nil
}

// Get implements Store.
func (s SecretsManager) Get(ctx context.Context, ref string) (connectors.Secrets, error) {
	if err := checkARN(ref); err != nil {
		return nil, err
	}
	out, err := s.Client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(ref)})
	var nf *smtypes.ResourceNotFoundException
	if errors.As(err, &nf) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if out.SecretString == nil {
		return nil, errors.New("the secret has no value")
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(*out.SecretString), &m); err != nil {
		return nil, errors.New("the secret is damaged")
	}
	return connectors.Secrets(m), nil
}

// Delete implements Store.
func (s SecretsManager) Delete(ctx context.Context, ref string) error {
	if err := checkARN(ref); err != nil {
		return err
	}
	_, err := s.Client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:             aws.String(ref),
		RecoveryWindowInDays: aws.Int64(RecoveryWindowDays),
	})
	var nf *smtypes.ResourceNotFoundException
	if errors.As(err, &nf) {
		return nil
	}
	return err
}
