package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var ErrNotFound = errors.New("api key not found")
var ErrUnauthorized = errors.New("unauthorized")

type Service struct {
	repo      Repository
	masterKey string // from config; empty = master key disabled
}

func NewService(repo Repository, masterKey string) *Service {
	return &Service{repo: repo, masterKey: masterKey}
}

// Create generates a new API key, stores its hash, and returns the raw key (shown once).
func (s *Service) Create(ctx context.Context, name string) (*APIKey, string, error) {
	raw, err := generateKey()
	if err != nil {
		return nil, "", err
	}
	hash := hashKey(raw)
	prefix := raw[:8]
	k, err := s.repo.Create(ctx, name, prefix, hash)
	if err != nil {
		return nil, "", err
	}
	return k, raw, nil
}

// Validate checks the key against master key first, then DB.
func (s *Service) Validate(ctx context.Context, key string) bool {
	if s.masterKey != "" && key == s.masterKey {
		return true
	}
	hash := hashKey(key)
	_, err := s.repo.GetByHash(ctx, hash)
	return err == nil
}

func (s *Service) List(ctx context.Context) ([]*APIKey, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// --- helpers ---

func generateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
