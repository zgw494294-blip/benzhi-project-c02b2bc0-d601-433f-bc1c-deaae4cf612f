package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"blast-permit/internal/domain"
)

type Mutation struct {
	EventType string
	Details   map[string]any
	Response  any
}
type Mutator func(*domain.CaseFile) (Mutation, error)

func (s *Store) Create(ctx context.Context, file *domain.CaseFile, operation, key, role, name string, response any) (json.RawMessage, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer rollback(tx)
	if raw, ok, err := readIdempotency(ctx, tx, operation, key); err != nil {
		return nil, false, err
	} else if ok {
		return raw, true, nil
	}
	if err = insertCase(ctx, tx, file); err != nil {
		return nil, false, err
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return nil, false, err
	}
	if err = appendAudit(ctx, tx, file.Case.CaseID, "case.created", role, name, map[string]any{"siteName": file.Case.SiteName}, file.Case.CreatedAt); err != nil {
		return nil, false, err
	}
	if err = writeIdempotency(ctx, tx, operation, key, file.Case.CaseID, raw, file.Case.CreatedAt); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return raw, false, nil
}

func (s *Store) Mutate(ctx context.Context, caseID string, expected int64, operation, key, role, name string, mutate Mutator) (json.RawMessage, bool, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, false, err
	}
	defer rollback(tx)
	if raw, ok, err := readIdempotency(context.Background(), tx, operation, key); err != nil {
		return nil, false, err
	} else if ok {
		return raw, true, nil
	}
	file, err := loadCaseTx(context.Background(), tx, caseID)
	if err != nil {
		return nil, false, err
	}
	if file.Case.Version != expected {
		return nil, false, domain.NewError(domain.CodeConflict, "expectedVersion=%d 与当前版本 %d 不一致", expected, file.Case.Version)
	}
	if !file.Case.State.Mutable() {
		return nil, false, domain.NewError(domain.CodeFrozen, "案卷已冻结，禁止业务修改")
	}
	m, err := mutate(file)
	if err != nil {
		return nil, false, err
	}
	if file.Case.Version != expected+1 {
		return nil, false, domain.NewError(domain.CodeCorrupt, "变更未正确推进聚合版本")
	}
	if err = saveCaseTx(context.Background(), tx, file, expected); err != nil {
		return nil, false, err
	}
	raw, err := json.Marshal(m.Response)
	if err != nil {
		return nil, false, err
	}
	if err = appendAudit(context.Background(), tx, caseID, m.EventType, role, name, m.Details, file.Case.UpdatedAt); err != nil {
		return nil, false, err
	}
	if err = writeIdempotency(context.Background(), tx, operation, key, caseID, raw, time.Now().UTC()); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return raw, false, nil
}

func scanNotFound(err error, subject string) error {
	if err == sql.ErrNoRows {
		return domain.NewError(domain.CodeNotFound, "%s 不存在", subject)
	}
	return err
}
