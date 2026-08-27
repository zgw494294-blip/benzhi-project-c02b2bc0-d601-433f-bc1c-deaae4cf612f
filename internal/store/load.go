package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"blast-permit/internal/domain"
)

func loadCaseTx(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, caseID string) (*domain.CaseFile, error) {
	f := &domain.CaseFile{}
	var created, updated string
	err := q.QueryRowContext(ctx, "SELECT case_id,site_name,work_zone,state,current_revision_id,version,created_at,updated_at FROM blast_cases WHERE case_id=?", caseID).Scan(&f.Case.CaseID, &f.Case.SiteName, &f.Case.WorkZone, &f.Case.State, &f.Case.CurrentRevisionID, &f.Case.Version, &created, &updated)
	if err != nil {
		return nil, scanNotFound(err, "案卷")
	}
	f.Case.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	f.Case.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	rows, err := q.QueryContext(ctx, "SELECT target_id,target_type,name,distance_m,allowed_ppv,baseline_ppv,measurement_note FROM protected_targets WHERE case_id=? ORDER BY target_id", caseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t domain.ProtectedTarget
		t.CaseID = caseID
		if err = rows.Scan(&t.TargetID, &t.TargetType, &t.Name, &t.DistanceMeters, &t.AllowedPpvMmPerSec, &t.BaselinePpvMmPerSec, &t.MeasurementNote); err != nil {
			rows.Close()
			return nil, err
		}
		f.Targets = append(f.Targets, t)
	}
	rows.Close()
	rows, err = q.QueryContext(ctx, "SELECT revision_id,revision_number,hole_pattern,max_charge,delay_sequence_json,initiation_direction,propagation_k,propagation_alpha,remediation_note,submitted_at,parent_revision_id,diff_json,review_comparison_json FROM design_revisions WHERE case_id=? ORDER BY revision_number", caseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r domain.DesignRevision
		var delays, at, diff, comparison string
		r.CaseID = caseID
		if err = rows.Scan(&r.RevisionID, &r.RevisionNumber, &r.HolePattern, &r.MaxChargePerDelayKg, &delays, &r.InitiationDirection, &r.PropagationK, &r.PropagationAlpha, &r.RemediationNote, &at, &r.ParentRevisionID, &diff, &comparison); err != nil {
			rows.Close()
			return nil, err
		}
		json.Unmarshal([]byte(delays), &r.DelaySequenceMs)
		json.Unmarshal([]byte(diff), &r.Diff)
		json.Unmarshal([]byte(comparison), &r.ReviewComparison)
		r.SubmittedAt, _ = time.Parse(time.RFC3339Nano, at)
		f.Revisions = append(f.Revisions, r)
	}
	rows.Close()
	rows, err = q.QueryContext(ctx, "SELECT assessment_id,revision_id,input_digest,predicted_json,findings_json,blocking_ids_json,formula_version,calculated_at,allowed_charges_json,input_summary_json,control_target_id,control_charge,charge_margin,charge_margin_percent FROM assessment_snapshots WHERE case_id=? ORDER BY calculated_at", caseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var a domain.AssessmentSnapshot
		var predicted, findings, blocking, at, allowed, inputSummary string
		var controlCharge, chargeMargin, chargeMarginPercent sql.NullFloat64
		if err = rows.Scan(&a.AssessmentID, &a.RevisionID, &a.InputDigest, &predicted, &findings, &blocking, &a.FormulaVersion, &at, &allowed, &inputSummary, &a.ControlTargetID, &controlCharge, &chargeMargin, &chargeMarginPercent); err != nil {
			rows.Close()
			return nil, err
		}
		json.Unmarshal([]byte(predicted), &a.PredictedTargets)
		json.Unmarshal([]byte(findings), &a.Findings)
		json.Unmarshal([]byte(blocking), &a.BlockingFindingIDs)
		json.Unmarshal([]byte(allowed), &a.AllowedCharges)
		json.Unmarshal([]byte(inputSummary), &a.InputSummary)
		a.ControlChargeKg = nullableFloat(controlCharge)
		a.ChargeMarginKg = nullableFloat(chargeMargin)
		a.ChargeMarginPercent = nullableFloat(chargeMarginPercent)
		a.CalculatedAt, _ = time.Parse(time.RFC3339Nano, at)
		f.Assessments = append(f.Assessments, a)
	}
	rows.Close()
	var baselineJSON string
	err = q.QueryRowContext(ctx, "SELECT summary_json FROM baseline_confirmations WHERE case_id=?", caseID).Scan(&baselineJSON)
	if err == nil {
		var confirmation domain.BaselineConfirmation
		if err = json.Unmarshal([]byte(baselineJSON), &confirmation); err != nil {
			return nil, err
		}
		f.BaselineConfirmation = &confirmation
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	rows, err = q.QueryContext(ctx, "SELECT transition_json FROM finding_transitions WHERE case_id=? ORDER BY rowid", caseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw string
		var transition domain.FindingTransition
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &transition); err != nil {
			rows.Close()
			return nil, err
		}
		f.FindingHistory = append(f.FindingHistory, transition)
	}
	rows.Close()
	rows, err = q.QueryContext(ctx, "SELECT round_json FROM review_rounds WHERE case_id=? ORDER BY review_round", caseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw string
		var review domain.ReviewRound
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &review); err != nil {
			rows.Close()
			return nil, err
		}
		f.Reviews = append(f.Reviews, review)
	}
	rows.Close()
	var p domain.MonitoringPlan
	var points, reviewed, validation string
	err = q.QueryRowContext(ctx, "SELECT plan_id,sensor_points_json,sample_rate_hz,trigger_ppv,evacuation_rule,remaining_risk,review_decision,reviewed_by,reviewed_at,revision_id,assessment_id,validation_json FROM monitoring_plans WHERE case_id=?", caseID).Scan(&p.PlanID, &points, &p.SampleRateHz, &p.TriggerPpvMmPerSec, &p.EvacuationRule, &p.RemainingRisk, &p.ReviewDecision, &p.ReviewedBy, &reviewed, &p.RevisionID, &p.AssessmentID, &validation)
	if err == nil {
		p.CaseID = caseID
		json.Unmarshal([]byte(points), &p.SensorPoints)
		json.Unmarshal([]byte(validation), &p.Validation)
		p.ReviewedAt, _ = time.Parse(time.RFC3339Nano, reviewed)
		f.MonitoringPlan = &p
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	var permit domain.IgnitionPermit
	var issued, valid string
	var components string
	err = q.QueryRowContext(ctx, "SELECT permit_number,frozen_revision_id,evidence_digest,verification_digest,issued_by,issued_at,valid_until,verification_status,frozen_components_json,frozen_audit_sequence,frozen_audit_head_digest FROM ignition_permits WHERE case_id=?", caseID).Scan(&permit.PermitNumber, &permit.FrozenRevisionID, &permit.EvidenceDigest, &permit.VerificationDigest, &permit.IssuedBy, &issued, &valid, &permit.VerificationStatus, &components, &permit.FrozenAuditSequence, &permit.FrozenAuditHeadDigest)
	if err == nil {
		permit.CaseID = caseID
		permit.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued)
		permit.ValidUntil, _ = time.Parse(time.RFC3339Nano, valid)
		json.Unmarshal([]byte(components), &permit.FrozenComponents)
		f.Permit = &permit
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	f.AuditState, err = loadAuditState(ctx, q, caseID)
	if err != nil {
		return nil, err
	}
	if assessment := f.CurrentAssessment(); assessment != nil {
		for _, finding := range assessment.Findings {
			if finding.Blocking {
				f.CurrentPendingFindings = append(f.CurrentPendingFindings, finding)
			}
		}
	}
	if review := f.CurrentReview(); review != nil && review.Decision == "reject" {
		f.CurrentReviewReasons = append(f.CurrentReviewReasons, review.Reasons...)
	}
	return f, nil
}

func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	out := value.Float64
	return &out
}

func loadAuditState(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, caseID string) (domain.AuditChainState, error) {
	state := domain.AuditChainState{Continuous: true, Digests: map[int64]string{}, EventTypes: map[int64]string{}}
	rows, err := q.QueryContext(ctx, "SELECT sequence,event_type,actor_role,actor_name,details_json,digest FROM audit_events WHERE case_id=? ORDER BY sequence", caseID)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	previous := ""
	var want int64 = 1
	for rows.Next() {
		var sequence int64
		var eventType, role, name, detailsJSON, digest string
		if err = rows.Scan(&sequence, &eventType, &role, &name, &detailsJSON, &digest); err != nil {
			return state, err
		}
		details := map[string]any{}
		if err = json.Unmarshal([]byte(detailsJSON), &details); err != nil {
			state.Continuous = false
		}
		if sequence != want || digest != domain.AuditDigest(previous, sequence, eventType, role, name, details) {
			state.Continuous = false
		}
		state.Digests[sequence] = digest
		state.EventTypes[sequence] = eventType
		state.Sequence = sequence
		state.HeadDigest = digest
		previous = digest
		want++
	}
	return state, rows.Err()
}
