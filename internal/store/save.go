package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"blast-permit/internal/domain"
)

func insertCase(ctx context.Context, tx *sql.Tx, f *domain.CaseFile) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO blast_cases(case_id,site_name,work_zone,state,current_revision_id,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)", f.Case.CaseID, f.Case.SiteName, f.Case.WorkZone, f.Case.State, f.Case.CurrentRevisionID, f.Case.Version, f.Case.CreatedAt.Format(time.RFC3339Nano), f.Case.UpdatedAt.Format(time.RFC3339Nano))
	return err
}
func saveCaseTx(ctx context.Context, tx *sql.Tx, f *domain.CaseFile, expected int64) error {
	res, err := tx.ExecContext(ctx, "UPDATE blast_cases SET state=?,current_revision_id=?,version=?,updated_at=? WHERE case_id=? AND version=?", f.Case.State, f.Case.CurrentRevisionID, f.Case.Version, f.Case.UpdatedAt.Format(time.RFC3339Nano), f.Case.CaseID, expected)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return domain.NewError(domain.CodeConflict, "案卷版本在事务期间发生变化")
	}
	for _, t := range f.Targets {
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO protected_targets(target_id,case_id,target_type,name,distance_m,allowed_ppv,baseline_ppv,measurement_note) VALUES(?,?,?,?,?,?,?,?)", t.TargetID, t.CaseID, t.TargetType, t.Name, t.DistanceMeters, t.AllowedPpvMmPerSec, t.BaselinePpvMmPerSec, t.MeasurementNote)
		if err != nil {
			return err
		}
	}
	for _, r := range f.Revisions {
		delays, _ := json.Marshal(r.DelaySequenceMs)
		diff, comparison := jsonText(r.Diff), jsonText(r.ReviewComparison)
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO design_revisions(revision_id,case_id,revision_number,hole_pattern,max_charge,delay_sequence_json,initiation_direction,propagation_k,propagation_alpha,remediation_note,submitted_at,parent_revision_id,diff_json,review_comparison_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)", r.RevisionID, r.CaseID, r.RevisionNumber, r.HolePattern, r.MaxChargePerDelayKg, string(delays), r.InitiationDirection, r.PropagationK, r.PropagationAlpha, r.RemediationNote, r.SubmittedAt.Format(time.RFC3339Nano), r.ParentRevisionID, diff, comparison)
		if err != nil {
			return err
		}
	}
	for _, a := range f.Assessments {
		pred, _ := json.Marshal(a.PredictedTargets)
		find, _ := json.Marshal(a.Findings)
		block, _ := json.Marshal(a.BlockingFindingIDs)
		allowed, _ := json.Marshal(a.AllowedCharges)
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO assessment_snapshots(assessment_id,case_id,revision_id,input_digest,predicted_json,findings_json,blocking_ids_json,formula_version,calculated_at,allowed_charges_json,input_summary_json,control_target_id,control_charge,charge_margin,charge_margin_percent) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", a.AssessmentID, f.Case.CaseID, a.RevisionID, a.InputDigest, string(pred), string(find), string(block), a.FormulaVersion, a.CalculatedAt.Format(time.RFC3339Nano), string(allowed), jsonText(a.InputSummary), a.ControlTargetID, a.ControlChargeKg, a.ChargeMarginKg, a.ChargeMarginPercent)
		if err != nil {
			return err
		}
	}
	if f.BaselineConfirmation != nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO baseline_confirmations(case_id,summary_json) VALUES(?,?) ON CONFLICT(case_id) DO UPDATE SET summary_json=excluded.summary_json", f.Case.CaseID, jsonText(f.BaselineConfirmation))
		if err != nil {
			return err
		}
	}
	for _, transition := range f.FindingHistory {
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO finding_transitions(case_id,revision_id,from_finding_id,to_finding_id,status,transition_json) VALUES(?,?,?,?,?,?)", f.Case.CaseID, transition.RevisionID, transition.FromFindingID, transition.ToFindingID, transition.Status, jsonText(transition))
		if err != nil {
			return err
		}
	}
	for _, review := range f.Reviews {
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO review_rounds(case_id,review_round,round_json) VALUES(?,?,?)", f.Case.CaseID, review.Round, jsonText(review))
		if err != nil {
			return err
		}
	}
	if p := f.MonitoringPlan; p != nil {
		points, _ := json.Marshal(p.SensorPoints)
		_, err = tx.ExecContext(ctx, "INSERT INTO monitoring_plans(plan_id,case_id,sensor_points_json,sample_rate_hz,trigger_ppv,evacuation_rule,remaining_risk,review_decision,reviewed_by,reviewed_at,revision_id,assessment_id,validation_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(case_id) DO UPDATE SET plan_id=excluded.plan_id,sensor_points_json=excluded.sensor_points_json,sample_rate_hz=excluded.sample_rate_hz,trigger_ppv=excluded.trigger_ppv,evacuation_rule=excluded.evacuation_rule,remaining_risk=excluded.remaining_risk,review_decision=excluded.review_decision,reviewed_by=excluded.reviewed_by,reviewed_at=excluded.reviewed_at,revision_id=excluded.revision_id,assessment_id=excluded.assessment_id,validation_json=excluded.validation_json", p.PlanID, p.CaseID, string(points), p.SampleRateHz, p.TriggerPpvMmPerSec, p.EvacuationRule, p.RemainingRisk, p.ReviewDecision, p.ReviewedBy, p.ReviewedAt.Format(time.RFC3339Nano), p.RevisionID, p.AssessmentID, jsonText(p.Validation))
		if err != nil {
			return err
		}
	}
	if p := f.Permit; p != nil {
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO ignition_permits(permit_number,case_id,frozen_revision_id,evidence_digest,verification_digest,issued_by,issued_at,valid_until,verification_status,frozen_components_json,frozen_audit_sequence,frozen_audit_head_digest) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", p.PermitNumber, p.CaseID, p.FrozenRevisionID, p.EvidenceDigest, p.VerificationDigest, p.IssuedBy, p.IssuedAt.Format(time.RFC3339Nano), p.ValidUntil.Format(time.RFC3339Nano), p.VerificationStatus, jsonText(p.FrozenComponents), p.FrozenAuditSequence, p.FrozenAuditHeadDigest)
		if err != nil {
			return err
		}
	}
	return nil
}
func jsonText(v any) string { b, _ := json.Marshal(v); return string(b) }
func dbError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("SQLite 写入失败: %w", err)
}
