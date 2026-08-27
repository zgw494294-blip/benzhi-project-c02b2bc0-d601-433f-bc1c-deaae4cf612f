package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"blast-permit/internal/domain"
)

func (s *Store) GetCase(ctx context.Context, caseID string) (*domain.CaseFile, error) {
	return loadCaseTx(ctx, s.db, caseID)
}
func (s *Store) GetPermit(ctx context.Context, number string) (*domain.IgnitionPermit, error) {
	var p domain.IgnitionPermit
	var issued, valid string
	var components string
	err := s.db.QueryRowContext(ctx, "SELECT permit_number,case_id,frozen_revision_id,evidence_digest,verification_digest,issued_by,issued_at,valid_until,verification_status,frozen_components_json,frozen_audit_sequence,frozen_audit_head_digest FROM ignition_permits WHERE permit_number=?", number).Scan(&p.PermitNumber, &p.CaseID, &p.FrozenRevisionID, &p.EvidenceDigest, &p.VerificationDigest, &p.IssuedBy, &issued, &valid, &p.VerificationStatus, &components, &p.FrozenAuditSequence, &p.FrozenAuditHeadDigest)
	if err != nil {
		return nil, scanNotFound(err, "许可")
	}
	p.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued)
	p.ValidUntil, _ = time.Parse(time.RFC3339Nano, valid)
	json.Unmarshal([]byte(components), &p.FrozenComponents)
	return &p, nil
}
func (s *Store) AuditCount(ctx context.Context, caseID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events WHERE case_id=?", caseID).Scan(&n)
	return n, err
}

var _ interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
} = (*sql.DB)(nil)
