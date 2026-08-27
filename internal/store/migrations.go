package store

import (
	"context"
	"database/sql"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS blast_cases(case_id TEXT PRIMARY KEY, site_name TEXT NOT NULL, work_zone TEXT NOT NULL, state TEXT NOT NULL, current_revision_id TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS protected_targets(target_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES blast_cases(case_id), target_type TEXT NOT NULL, name TEXT NOT NULL, distance_m REAL NOT NULL CHECK(distance_m>0), allowed_ppv REAL NOT NULL CHECK(allowed_ppv>0), baseline_ppv REAL NOT NULL CHECK(baseline_ppv>=0), measurement_note TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS design_revisions(revision_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES blast_cases(case_id), revision_number INTEGER NOT NULL, hole_pattern TEXT NOT NULL, max_charge REAL NOT NULL, delay_sequence_json TEXT NOT NULL, initiation_direction TEXT NOT NULL, propagation_k REAL NOT NULL, propagation_alpha REAL NOT NULL, remediation_note TEXT NOT NULL, submitted_at TEXT NOT NULL, UNIQUE(case_id,revision_number));
CREATE TABLE IF NOT EXISTS assessment_snapshots(assessment_id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES blast_cases(case_id), revision_id TEXT NOT NULL REFERENCES design_revisions(revision_id), input_digest TEXT NOT NULL, predicted_json TEXT NOT NULL, findings_json TEXT NOT NULL, blocking_ids_json TEXT NOT NULL, formula_version TEXT NOT NULL, calculated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS monitoring_plans(plan_id TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE REFERENCES blast_cases(case_id), sensor_points_json TEXT NOT NULL, sample_rate_hz INTEGER NOT NULL, trigger_ppv REAL NOT NULL, evacuation_rule TEXT NOT NULL, remaining_risk TEXT NOT NULL, review_decision TEXT NOT NULL, reviewed_by TEXT NOT NULL, reviewed_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS ignition_permits(permit_number TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE REFERENCES blast_cases(case_id), frozen_revision_id TEXT NOT NULL, evidence_digest TEXT NOT NULL, verification_digest TEXT NOT NULL, issued_by TEXT NOT NULL, issued_at TEXT NOT NULL, valid_until TEXT NOT NULL, verification_status TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS idempotency_results(operation TEXT NOT NULL, idempotency_key TEXT NOT NULL, case_id TEXT NOT NULL, response_json BLOB NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(operation,idempotency_key));
CREATE TABLE IF NOT EXISTS audit_events(case_id TEXT NOT NULL REFERENCES blast_cases(case_id), sequence INTEGER NOT NULL, event_type TEXT NOT NULL, actor_role TEXT NOT NULL, actor_name TEXT NOT NULL, details_json TEXT NOT NULL, created_at TEXT NOT NULL, digest TEXT NOT NULL, PRIMARY KEY(case_id,sequence));
CREATE INDEX IF NOT EXISTS idx_targets_case ON protected_targets(case_id);
CREATE INDEX IF NOT EXISTS idx_revisions_case ON design_revisions(case_id,revision_number);
CREATE INDEX IF NOT EXISTS idx_assessments_case ON assessment_snapshots(case_id,calculated_at);`,
	`ALTER TABLE design_revisions ADD COLUMN parent_revision_id TEXT NOT NULL DEFAULT '';
ALTER TABLE design_revisions ADD COLUMN diff_json TEXT NOT NULL DEFAULT '{"baseline":false,"changes":[],"affectedTargetIds":[],"recalculateAllTargets":false,"delayAssessmentAffected":false,"reviewAttention":[]}';
ALTER TABLE design_revisions ADD COLUMN review_comparison_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE assessment_snapshots ADD COLUMN allowed_charges_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE assessment_snapshots ADD COLUMN input_summary_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE assessment_snapshots ADD COLUMN control_target_id TEXT NOT NULL DEFAULT '';
ALTER TABLE assessment_snapshots ADD COLUMN control_charge REAL;
ALTER TABLE assessment_snapshots ADD COLUMN charge_margin REAL;
ALTER TABLE assessment_snapshots ADD COLUMN charge_margin_percent REAL;
ALTER TABLE monitoring_plans ADD COLUMN revision_id TEXT NOT NULL DEFAULT '';
ALTER TABLE monitoring_plans ADD COLUMN assessment_id TEXT NOT NULL DEFAULT '';
ALTER TABLE monitoring_plans ADD COLUMN validation_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE ignition_permits ADD COLUMN frozen_components_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE ignition_permits ADD COLUMN frozen_audit_sequence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ignition_permits ADD COLUMN frozen_audit_head_digest TEXT NOT NULL DEFAULT '';
CREATE TABLE baseline_confirmations(case_id TEXT PRIMARY KEY REFERENCES blast_cases(case_id), summary_json TEXT NOT NULL);
CREATE TABLE finding_transitions(case_id TEXT NOT NULL REFERENCES blast_cases(case_id), revision_id TEXT NOT NULL REFERENCES design_revisions(revision_id), from_finding_id TEXT NOT NULL, to_finding_id TEXT NOT NULL, status TEXT NOT NULL, transition_json TEXT NOT NULL, PRIMARY KEY(revision_id,from_finding_id,to_finding_id,status));
CREATE TABLE review_rounds(case_id TEXT NOT NULL REFERENCES blast_cases(case_id), review_round INTEGER NOT NULL, round_json TEXT NOT NULL, PRIMARY KEY(case_id,review_round));
CREATE INDEX idx_finding_transitions_case ON finding_transitions(case_id,revision_id);
CREATE INDEX idx_review_rounds_case ON review_rounds(case_id,review_round);`,
}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		return err
	}
	for i, script := range migrations {
		version := i + 1
		var exists int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=?", version).Scan(&exists)
		if err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		if _, err = tx.ExecContext(ctx, script); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,applied_at) VALUES(?,datetime('now'))", version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func rollback(tx *sql.Tx) { _ = tx.Rollback() }
