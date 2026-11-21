// Package credentials keeps connection credentials out of Postgres (ADR-016).
// Postgres holds only the reference a Store returns. Local encrypts with a key
// from the environment; SecretsManager keeps one AWS secret per connection.
package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/google/uuid"
)

// ErrNotFound means no secret has this reference.
var ErrNotFound = errors.New("secret not found")

// Store keeps one secret per connection.
type Store interface {
	// Put stores a connection's credentials and returns their reference.
	Put(ctx context.Context, connectionID string, s connectors.Secrets) (ref string, err error)
	// Get reads the credentials at ref.
	Get(ctx context.Context, ref string) (connectors.Secrets, error)
	// Delete removes the credentials at ref. A missing secret is not an error.
	Delete(ctx context.Context, ref string) error
}

// KeySize is the local key's length in bytes (AES-256).
const KeySize = 32

// ParseKey decodes a base64 key and checks its length. Its errors never hold
// the key.
func ParseKey(b64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, errors.New("the credential key is not valid base64")
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("the credential key must be %d bytes", KeySize)
	}
	return key, nil
}

const localPrefix = "local:"

// Local encrypts each connection's credentials with AES-256-GCM into one file.
// The reference is bound into the ciphertext, so a file moved to another
// reference does not decrypt.
type Local struct {
	dir  string
	aead cipher.AEAD
}

// NewLocal returns a Local store in dir, creating it.
func NewLocal(dir string, key []byte) (*Local, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("the credential key must be %d bytes", KeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Local{dir: dir, aead: aead}, nil
}

func (l *Local) path(ref string) (string, error) {
	id, ok := strings.CutPrefix(ref, localPrefix)
	if !ok {
		return "", errors.New("not a local secret reference")
	}
	u, err := uuid.Parse(id)
	if err != nil || u.String() != id {
		return "", errors.New("not a local secret reference")
	}
	return filepath.Join(l.dir, id+".secret"), nil
}

// Put implements Store.
func (l *Local) Put(_ context.Context, connectionID string, s connectors.Secrets) (string, error) {
	ref := localPrefix + connectionID
	p, err := l.path(ref)
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(map[string]string(s))
	if err != nil {
		return "", err
	}
	nonce := make([]byte, l.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := l.aead.Seal(nonce, nonce, plain, []byte(ref))
	tmp, err := os.CreateTemp(l.dir, ".tmp-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(sealed); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return "", err
	}
	return ref, nil
}

// Get implements Store.
func (l *Local) Get(_ context.Context, ref string) (connectors.Secrets, error) {
	p, err := l.path(ref)
	if err != nil {
		return nil, err
	}
	sealed, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	n := l.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("the secret file is damaged")
	}
	plain, err := l.aead.Open(nil, sealed[:n], sealed[n:], []byte(ref))
	if err != nil {
		return nil, errors.New("the secret does not decrypt with this key")
	}
	var s map[string]string
	if err := json.Unmarshal(plain, &s); err != nil {
		return nil, errors.New("the secret is damaged")
	}
	return connectors.Secrets(s), nil
}

// Delete implements Store.
func (l *Local) Delete(_ context.Context, ref string) error {
	p, err := l.path(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
