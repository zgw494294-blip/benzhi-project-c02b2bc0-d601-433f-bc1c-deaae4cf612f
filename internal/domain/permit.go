package domain

import "time"

func BuildPermitPrecheck(f CaseFile) PermitPrecheck {
	result := PermitPrecheck{CaseID: f.Case.CaseID, Components: EvidenceComponents(f), AuditSequence: f.AuditState.Sequence, AuditHeadDigest: f.AuditState.HeadDigest}
	markMismatch := func(name string) {
		for i := range result.Components {
			if result.Components[i].Name == name && result.Components[i].Status == "present" {
				result.Components[i].Status = "mismatch"
			}
		}
	}
	for _, component := range result.Components {
		if component.Status != "present" {
			result.Blockers = append(result.Blockers, EvidenceIssue{Code: "EVIDENCE_COMPONENT_MISSING", Component: component.Name, Message: "许可冻结证据组件缺失"})
		}
	}
	if f.Case.State != StateApproved {
		result.Blockers = append(result.Blockers, EvidenceIssue{Code: "CASE_NOT_APPROVED", Message: "案卷尚未处于 approved 状态"})
	}
	baseline := PrecheckBaseline(f.Case.CaseID, f.Targets)
	if f.BaselineConfirmation != nil {
		expected := NewBaselineConfirmation(baseline, f.BaselineConfirmation.ConfirmedAt)
		if !baseline.Ready || Digest(expected) != Digest(*f.BaselineConfirmation) {
			markMismatch("baseline")
			result.Blockers = append(result.Blockers, EvidenceIssue{Code: "BASELINE_EVIDENCE_MISMATCH", Component: "baseline", Message: "基线确认摘要与当前保护对象不一致"})
		}
	}
	revision, assessment := f.CurrentRevision(), f.CurrentAssessment()
	if revision != nil && (assessment == nil || assessment.RevisionID != revision.RevisionID) {
		markMismatch("assessment")
		result.Blockers = append(result.Blockers, EvidenceIssue{Code: "ASSESSMENT_REVISION_MISMATCH", Component: "assessment", Message: "当前评估与当前修订不对应"})
	}
	if assessment != nil && HasBlockers(*assessment) {
		result.Blockers = append(result.Blockers, EvidenceIssue{Code: "ASSESSMENT_HAS_BLOCKERS", Component: "assessment", Message: "当前评估仍存在阻断项"})
	}
	if revision != nil && assessment != nil {
		expected := Assess(*revision, f.Targets, assessment.CalculatedAt, assessment.AssessmentID)
		if Digest(expected) != Digest(*assessment) {
			markMismatch("assessment")
			result.Blockers = append(result.Blockers, EvidenceIssue{Code: "ASSESSMENT_EVIDENCE_MISMATCH", Component: "assessment", Message: "评估快照与当前修订输入复算结果不一致"})
		}
	}
	review := f.CurrentReview()
	if f.MonitoringPlan == nil || f.MonitoringPlan.ReviewDecision != "approve" || review == nil || review.Decision != "approve" || review.Plan.PlanID != f.MonitoringPlan.PlanID || review.RevisionID != f.Case.CurrentRevisionID || assessment == nil || review.AssessmentID != assessment.AssessmentID || f.MonitoringPlan.RevisionID != f.Case.CurrentRevisionID || f.MonitoringPlan.AssessmentID != assessment.AssessmentID || !review.Validation.Ready || !f.MonitoringPlan.Validation.Ready {
		if f.MonitoringPlan != nil {
			markMismatch("monitoringPlan")
		}
		result.Blockers = append(result.Blockers, EvidenceIssue{Code: "MONITORING_PLAN_NOT_APPROVED", Component: "monitoringPlan", Message: "缺少获批监测计划"})
	}
	if f.MonitoringPlan != nil && review != nil && assessment != nil {
		expected := BuildMonitoringValidation(*f.MonitoringPlan, f.Targets, *assessment)
		if Digest(expected) != Digest(f.MonitoringPlan.Validation) || Digest(review.Plan) != Digest(*f.MonitoringPlan) || Digest(review.Validation) != Digest(expected) {
			markMismatch("monitoringPlan")
			result.Blockers = append(result.Blockers, EvidenceIssue{Code: "MONITORING_EVIDENCE_MISMATCH", Component: "monitoringPlan", Message: "监测计划、复核轮次或联锁校验摘要不一致"})
		}
	}
	if !f.AuditState.Continuous {
		result.Blockers = append(result.Blockers, EvidenceIssue{Code: "AUDIT_CHAIN_BROKEN", Message: "审计时间线不连续或摘要不匹配"})
	}
	result.EvidenceDigest = TotalEvidenceDigest(f.Case.CaseID, result.Components)
	result.Ready = len(result.Blockers) == 0
	return result
}

func NewPermit(number string, f CaseFile, precheck PermitPrecheck, issuer string, now, validUntil time.Time) IgnitionPermit {
	evidence := precheck.EvidenceDigest
	verification := PermitRecordDigest(number, f.Case.CaseID, evidence, f.Case.CurrentRevisionID, now, validUntil, precheck.AuditSequence, precheck.AuditHeadDigest)
	return IgnitionPermit{PermitNumber: number, CaseID: f.Case.CaseID, FrozenRevisionID: f.Case.CurrentRevisionID, EvidenceDigest: evidence, VerificationDigest: verification, IssuedBy: issuer, IssuedAt: now, ValidUntil: validUntil, VerificationStatus: "valid", FrozenComponents: append([]EvidenceComponent(nil), precheck.Components...), FrozenAuditSequence: precheck.AuditSequence, FrozenAuditHeadDigest: precheck.AuditHeadDigest}
}
func VerifyPermit(p IgnitionPermit) bool {
	want := PermitRecordDigest(p.PermitNumber, p.CaseID, p.EvidenceDigest, p.FrozenRevisionID, p.IssuedAt, p.ValidUntil, p.FrozenAuditSequence, p.FrozenAuditHeadDigest)
	return p.VerificationDigest == want && p.VerificationStatus == "valid"
}
