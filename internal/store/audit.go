package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"blast-permit/internal/domain"
)

func appendAudit(ctx context.Context, tx *sql.Tx, caseID, eventType, role, name string, details map[string]any, at time.Time) error {
	var sequence int64
	var previous string
	err := tx.QueryRowContext(ctx, "SELECT sequence,digest FROM audit_events WHERE case_id=? ORDER BY sequence DESC LIMIT 1", caseID).Scan(&sequence, &previous)
	if err == sql.ErrNoRows {
		sequence = 0
		previous = ""
	} else if err != nil {
		return err
	}
	sequence++
	raw, _ := json.Marshal(details)
	normalized := map[string]any{}
	_ = json.Unmarshal(raw, &normalized)
	digest := domain.AuditDigest(previous, sequence, eventType, role, name, normalized)
	_, err = tx.ExecContext(ctx, "INSERT INTO audit_events(case_id,sequence,event_type,actor_role,actor_name,details_json,created_at,digest) VALUES(?,?,?,?,?,?,?,?)", caseID, sequence, eventType, role, name, string(raw), at.Format(time.RFC3339Nano), digest)
	return err
}

func (s *Store) Audit(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT sequence,event_type,actor_role,actor_name,details_json,created_at,digest FROM audit_events WHERE case_id=? ORDER BY sequence", caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.AuditEvent{}
	previous := ""
	var want int64 = 1
	for rows.Next() {
		var e domain.AuditEvent
		var details, at string
		if err = rows.Scan(&e.Sequence, &e.EventType, &e.ActorRole, &e.ActorName, &details, &at, &e.Digest); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(details), &e.Details)
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, at)
		if e.Sequence != want || e.Digest != domain.AuditDigest(previous, e.Sequence, e.EventType, e.ActorRole, e.ActorName, e.Details) {
			return nil, domain.NewError(domain.CodeCorrupt, "审计时间线在序号 %d 处不连续或摘要不匹配", want)
		}
		previous = e.Digest
		want++
		events = append(events, e)
	}
	if len(events) == 0 {
		if _, err = s.GetCase(ctx, caseID); err != nil {
			return nil, err
		}
	}
	return events, rows.Err()
}
