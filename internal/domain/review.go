package domain

import (
	"sort"
	"strings"
)

func BuildMonitoringValidation(plan MonitoringPlan, targets []ProtectedTarget, assessment AssessmentSnapshot) MonitoringValidation {
	result := MonitoringValidation{}
	predictions := map[string]PredictedTarget{}
	for _, predicted := range assessment.PredictedTargets {
		predictions[predicted.TargetID] = predicted
	}
	covered := map[string][]string{}
	for _, sensor := range plan.SensorPoints {
		covered[sensor.TargetID] = append(covered[sensor.TargetID], strings.TrimSpace(sensor.Name))
	}
	for _, target := range targets {
		reasons := []string{}
		if target.TargetID == assessment.ControlTargetID {
			reasons = append(reasons, "control_target")
		}
		if predicted, ok := predictions[target.TargetID]; ok && target.AllowedPpvMmPerSec > 0 && predicted.MarginMmPerSec/target.AllowedPpvMmPerSec <= 0.2 {
			reasons = append(reasons, "critical_margin")
		}
		names := covered[target.TargetID]
		sort.Strings(names)
		entry := CoverageEntry{TargetID: target.TargetID, Required: len(reasons) > 0, Reasons: reasons, SensorNames: names, Covered: len(names) > 0}
		result.CoverageMatrix = append(result.CoverageMatrix, entry)
		if entry.Required && !entry.Covered {
			result.UncoveredTargetIDs = append(result.UncoveredTargetIDs, target.TargetID)
		}
		if entry.Covered && plan.TriggerPpvMmPerSec > target.AllowedPpvMmPerSec {
			result.ThresholdIssues = append(result.ThresholdIssues, MonitoringIssue{Code: "TRIGGER_ABOVE_ALLOWED_PPV", TargetID: target.TargetID, Message: "触发阈值高于已覆盖对象允许振速", Limit: target.AllowedPpvMmPerSec, Actual: plan.TriggerPpvMmPerSec})
		}
		if predicted, ok := predictions[target.TargetID]; ok && entry.Covered && predicted.PredictedPpvMmPerSec >= plan.TriggerPpvMmPerSec*0.8 {
			result.Attention = append(result.Attention, MonitoringIssue{Code: "PREDICTION_NEAR_TRIGGER", TargetID: target.TargetID, Message: "预测振速已接近监测触发阈值", Limit: plan.TriggerPpvMmPerSec, Actual: predicted.PredictedPpvMmPerSec})
		}
	}
	sort.Slice(result.CoverageMatrix, func(i, j int) bool { return result.CoverageMatrix[i].TargetID < result.CoverageMatrix[j].TargetID })
	sort.Strings(result.UncoveredTargetIDs)
	result.Ready = len(result.UncoveredTargetIDs) == 0 && len(result.ThresholdIssues) == 0
	return result
}

func ValidateReviewReasons(reasons []ReviewReason, targets []ProtectedTarget) error {
	if len(reasons) == 0 {
		return NewError(CodeReviewReasons, "退回决定必须包含至少一个结构化原因")
	}
	knownTargets := map[string]bool{}
	for _, target := range targets {
		knownTargets[target.TargetID] = true
	}
	for i, reason := range reasons {
		if strings.TrimSpace(reason.Category) == "" || strings.TrimSpace(reason.Description) == "" || strings.TrimSpace(reason.RequiredChange) == "" {
			return NewDetailedError(CodeReviewReasons, map[string]any{"itemIndex": i + 1}, "退回原因必须包含 category、description 和 requiredChange")
		}
		if reason.TargetID == "" && strings.TrimSpace(reason.Parameter) == "" {
			return NewDetailedError(CodeReviewReasons, map[string]any{"itemIndex": i + 1}, "退回原因必须关联 targetId 或 parameter")
		}
		if reason.TargetID != "" && !knownTargets[reason.TargetID] {
			return NewDetailedError(CodeReviewReasons, map[string]any{"itemIndex": i + 1, "targetId": reason.TargetID}, "退回原因引用了非本案保护对象")
		}
	}
	return nil
}

func LastRejectedReview(reviews []ReviewRound) *ReviewRound {
	for i := len(reviews) - 1; i >= 0; i-- {
		if reviews[i].Decision == "reject" {
			return &reviews[i]
		}
	}
	return nil
}
