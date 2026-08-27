package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

var orderedTargetTypes = []string{"building", "pipeline", "equipment"}

func PrecheckBaseline(caseID string, targets []ProtectedTarget) BaselinePrecheck {
	result := BaselinePrecheck{CaseID: caseID, TypeCounts: map[string]int{"building": 0, "pipeline": 0, "equipment": 0}}
	for _, target := range targets {
		result.TypeCounts[target.TargetType]++
		if !finite(target.DistanceMeters) || target.DistanceMeters <= 0 {
			result.Issues = append(result.Issues, BaselineIssue{Code: "INVALID_DISTANCE", TargetID: target.TargetID, Field: "distanceMeters", Message: "距离必须为有限正数"})
		}
		if !finite(target.AllowedPpvMmPerSec) || target.AllowedPpvMmPerSec <= 0 {
			result.Issues = append(result.Issues, BaselineIssue{Code: "INVALID_ALLOWED_PPV", TargetID: target.TargetID, Field: "allowedPpvMmPerSec", Message: "允许振速必须为有限正数"})
		}
		if !finite(target.BaselinePpvMmPerSec) || target.BaselinePpvMmPerSec < 0 || target.BaselinePpvMmPerSec >= target.AllowedPpvMmPerSec {
			result.Issues = append(result.Issues, BaselineIssue{Code: "INVALID_BASELINE_PPV", TargetID: target.TargetID, Field: "baselinePpvMmPerSec", Message: "基线振速必须非负且小于允许振速"})
		}
		if strings.TrimSpace(target.MeasurementNote) == "" {
			result.Issues = append(result.Issues, BaselineIssue{Code: "MEASUREMENT_NOTE_MISSING", TargetID: target.TargetID, Field: "measurementNote", Message: "缺少基线测量说明"})
		}
		if target.AllowedPpvMmPerSec > 0 && finite(target.AllowedPpvMmPerSec) && finite(target.BaselinePpvMmPerSec) {
			remaining := round(target.AllowedPpvMmPerSec-target.BaselinePpvMmPerSec, 3)
			ratio := round(remaining/target.AllowedPpvMmPerSec, 6)
			level := "sufficient"
			if ratio <= 0.2 {
				level = "critical"
			} else if ratio <= 0.5 {
				level = "attention"
			}
			result.TargetMargins = append(result.TargetMargins, BaselineTargetMargin{TargetID: target.TargetID, RemainingPpvMmPerSec: remaining, RemainingRatio: ratio, RiskLevel: level})
		}
	}
	for _, targetType := range orderedTargetTypes {
		if result.TypeCounts[targetType] == 0 {
			result.MissingTypes = append(result.MissingTypes, targetType)
			result.Issues = append(result.Issues, BaselineIssue{Code: "TARGET_TYPE_MISSING", Field: "targetType", Message: "缺少 " + targetType + " 类型保护对象"})
		}
	}
	sort.Slice(result.TargetMargins, func(i, j int) bool {
		if result.TargetMargins[i].RemainingRatio == result.TargetMargins[j].RemainingRatio {
			return result.TargetMargins[i].TargetID < result.TargetMargins[j].TargetID
		}
		return result.TargetMargins[i].RemainingRatio < result.TargetMargins[j].RemainingRatio
	})
	for i := range result.TargetMargins {
		result.TargetMargins[i].ControlOrder = i + 1
	}
	if len(result.TargetMargins) > 0 {
		result.ControlTargetID = result.TargetMargins[0].TargetID
	}
	result.Ready = len(result.Issues) == 0
	return result
}

func NewBaselineConfirmation(precheck BaselinePrecheck, at time.Time) BaselineConfirmation {
	riskCounts := map[string]int{"sufficient": 0, "attention": 0, "critical": 0}
	for _, margin := range precheck.TargetMargins {
		riskCounts[margin.RiskLevel]++
	}
	counts := map[string]int{}
	for key, value := range precheck.TypeCounts {
		counts[key] = value
	}
	return BaselineConfirmation{ConfirmedAt: at, TypeCounts: counts, TargetCount: len(precheck.TargetMargins), ControlTargetID: precheck.ControlTargetID, RiskCounts: riskCounts}
}

func round(value float64, digits int) float64 {
	factor := math.Pow10(digits)
	return math.Round(value*factor) / factor
}
