package domain

import (
	"math"
	"time"
)

const FormulaVersion = "scaled-distance-v1"

func Assess(revision DesignRevision, targets []ProtectedTarget, now time.Time, assessmentID string) AssessmentSnapshot {
	a := AssessmentSnapshot{AssessmentID: assessmentID, RevisionID: revision.RevisionID, FormulaVersion: FormulaVersion, CalculatedAt: now}
	if revision.PropagationK <= 0 || revision.PropagationAlpha <= 0 || !finite(revision.PropagationK) || !finite(revision.PropagationAlpha) {
		a.Findings = append(a.Findings, NewFinding(revision.RevisionID, "PROPAGATION_PARAMETER_MISSING", "", "缺少完整场地传播参数，无法形成可信预测"))
		for _, t := range targets {
			a.PredictedTargets = append(a.PredictedTargets, PredictedTarget{TargetID: t.TargetID, Pass: false})
			a.AllowedCharges = append(a.AllowedCharges, AllowedChargeResult{TargetID: t.TargetID, Status: "unavailable"})
		}
		for _, f := range a.Findings {
			a.BlockingFindingIDs = append(a.BlockingFindingIDs, f.FindingID)
		}
		finalizeAssessmentInput(&a, revision, targets)
		return a
	}
	parameterResultInvalid := false
	for _, t := range targets {
		pred := revision.PropagationK * math.Pow(math.Sqrt(revision.MaxChargePerDelayKg)/t.DistanceMeters, revision.PropagationAlpha)
		if !finite(pred) {
			a.PredictedTargets = append(a.PredictedTargets, PredictedTarget{TargetID: t.TargetID, Pass: false})
			a.AllowedCharges = append(a.AllowedCharges, AllowedChargeResult{TargetID: t.TargetID, Status: "unavailable"})
			parameterResultInvalid = true
			continue
		}
		pred = round(pred, 3)
		margin := round(t.AllowedPpvMmPerSec-pred, 3)
		a.PredictedTargets = append(a.PredictedTargets, PredictedTarget{TargetID: t.TargetID, PredictedPpvMmPerSec: pred, MarginMmPerSec: margin, Pass: margin >= 0})
		if margin < 0 {
			a.Findings = append(a.Findings, NewFinding(revision.RevisionID, "PPV_EXCEEDED", t.TargetID, "预测振速超过保护对象允许值"))
		}
		allowed := math.Pow(t.DistanceMeters*math.Pow(t.AllowedPpvMmPerSec/revision.PropagationK, 1/revision.PropagationAlpha), 2)
		if !finite(allowed) || allowed < 0 {
			a.AllowedCharges = append(a.AllowedCharges, AllowedChargeResult{TargetID: t.TargetID, Status: "unavailable"})
			parameterResultInvalid = true
			continue
		}
		allowed = round(allowed, 6)
		if allowed <= 0 {
			a.AllowedCharges = append(a.AllowedCharges, AllowedChargeResult{TargetID: t.TargetID, Status: "unavailable"})
			parameterResultInvalid = true
			continue
		}
		a.AllowedCharges = append(a.AllowedCharges, AllowedChargeResult{TargetID: t.TargetID, Status: "available", MaxAllowedChargeKg: &allowed})
	}
	if parameterResultInvalid {
		a.Findings = append(a.Findings, NewFinding(revision.RevisionID, "PROPAGATION_PARAMETER_INVALID", "", "传播参数计算产生非有限或不可反算结果"))
	}
	controlIndex := -1
	for i, result := range a.AllowedCharges {
		if result.Status != "available" || result.MaxAllowedChargeKg == nil {
			continue
		}
		if controlIndex < 0 || *result.MaxAllowedChargeKg < *a.AllowedCharges[controlIndex].MaxAllowedChargeKg || (*result.MaxAllowedChargeKg == *a.AllowedCharges[controlIndex].MaxAllowedChargeKg && result.TargetID < a.AllowedCharges[controlIndex].TargetID) {
			controlIndex = i
		}
	}
	if controlIndex >= 0 {
		a.AllowedCharges[controlIndex].Control = true
		a.ControlTargetID = a.AllowedCharges[controlIndex].TargetID
		control := *a.AllowedCharges[controlIndex].MaxAllowedChargeKg
		margin := round(control-revision.MaxChargePerDelayKg, 6)
		percent := round(margin/control*100, 3)
		a.ControlChargeKg, a.ChargeMarginKg, a.ChargeMarginPercent = &control, &margin, &percent
	}
	if DelayAnomaly(revision.DelaySequenceMs) {
		a.Findings = append(a.Findings, NewFinding(revision.RevisionID, "DELAY_ANOMALY", "", "延期序列存在重复、倒序或过小间隔"))
	}
	for _, f := range a.Findings {
		if f.Blocking {
			a.BlockingFindingIDs = append(a.BlockingFindingIDs, f.FindingID)
		}
	}
	finalizeAssessmentInput(&a, revision, targets)
	return a
}

func finalizeAssessmentInput(assessment *AssessmentSnapshot, revision DesignRevision, targets []ProtectedTarget) {
	summary := AssessmentInputSummary{RevisionID: revision.RevisionID, MaxChargePerDelayKg: revision.MaxChargePerDelayKg, PropagationK: revision.PropagationK, PropagationAlpha: revision.PropagationAlpha, FormulaVersion: FormulaVersion, AllowedCharges: append([]AllowedChargeResult(nil), assessment.AllowedCharges...), ControlTargetID: assessment.ControlTargetID}
	for _, target := range targets {
		summary.Targets = append(summary.Targets, AssessmentTargetInput{TargetID: target.TargetID, DistanceMeters: target.DistanceMeters, AllowedPpvMmPerSec: target.AllowedPpvMmPerSec})
	}
	assessment.InputSummary = summary
	assessment.InputDigest = Digest(summary)
}

func DelayAnomaly(seq []int) bool {
	for i := 1; i < len(seq); i++ {
		if seq[i] <= seq[i-1] || seq[i]-seq[i-1] < 5 {
			return true
		}
	}
	return false
}
