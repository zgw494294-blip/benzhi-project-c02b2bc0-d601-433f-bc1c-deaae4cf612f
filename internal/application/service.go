package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

type Service struct {
	store *store.Store
	now   func() time.Time
}

func New(s *store.Store) *Service {
	return &Service{store: s, now: func() time.Time { return time.Now().UTC() }}
}
func NewWithClock(s *store.Store, now func() time.Time) *Service { return &Service{store: s, now: now} }
func newID(prefix string) string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}
func decodeResult[T any](raw json.RawMessage) (T, error) {
	var out T
	err := json.Unmarshal(raw, &out)
	return out, err
}
func requireMutation(expected int64, key string) error {
	if expected < 1 {
		return domain.NewError(domain.CodeValidation, "expectedVersion 必须大于 0")
	}
	if len(key) < 8 || len(key) > 128 {
		return domain.NewError(domain.CodeValidation, "idempotencyKey 长度必须为 8 到 128")
	}
	return nil
}
func requireKey(key string) error {
	if len(key) < 8 || len(key) > 128 {
		return domain.NewError(domain.CodeValidation, "idempotencyKey 长度必须为 8 到 128")
	}
	return nil
}
func operation(name, caseID string) string { return fmt.Sprintf("%s:%s", name, caseID) }

type Repository interface {
	GetCase(context.Context, string) (*domain.CaseFile, error)
}
