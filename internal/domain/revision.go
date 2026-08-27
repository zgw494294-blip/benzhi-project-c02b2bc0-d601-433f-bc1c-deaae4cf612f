package domain

import (
	"reflect"
	"sort"
	"strings"
)

func NormalizeRevision(r DesignRevision) DesignRevision {
	r.HolePattern = strings.TrimSpace(r.HolePattern)
	r.InitiationDirection = strings.TrimSpace(r.InitiationDirection)
	r.RemediationNote = strings.TrimSpace(r.RemediationNote)
	r.DelaySequenceMs = append([]int(nil), r.DelaySequenceMs...)
	return r
}

func BuildRevisionDiff(previous *DesignRevision, current DesignRevision, targets []ProtectedTarget) (RevisionDiff, error) {
	diff := RevisionDiff{Baseline: previous == nil}
	add := func(field string, oldValue, newValue any, delta *float64) {
		diff.Changes = append(diff.Changes, ParameterChange{Field: field, OldValue: oldValue, NewValue: newValue, NumericDelta: delta})
	}
	if previous == nil {
		add("holePattern", nil, current.HolePattern, nil)
		add("maxChargePerDelayKg", nil, current.MaxChargePerDelayKg, nil)
		add("delaySequenceMs", nil, current.DelaySequenceMs, nil)
		add("initiationDirection", nil, current.InitiationDirection, nil)
		add("propagationK", nil, current.PropagationK, nil)
		add("propagationAlpha", nil, current.PropagationAlpha, nil)
		diff.RecalculateAllTargets = true
		diff.DelayAssessmentAffected = true
		diff.ReviewAttention = []string{"holePattern", "initiationDirection"}
	} else {
		if previous.HolePattern != current.HolePattern {
			add("holePattern", previous.HolePattern, current.HolePattern, nil)
			diff.ReviewAttention = append(diff.ReviewAttention, "holePattern")
		}
		if previous.MaxChargePerDelayKg != current.MaxChargePerDelayKg {
			delta := round(current.MaxChargePerDelayKg-previous.MaxChargePerDelayKg, 6)
			add("maxChargePerDelayKg", previous.MaxChargePerDelayKg, current.MaxChargePerDelayKg, &delta)
			diff.RecalculateAllTargets = true
		}
		if !reflect.DeepEqual(previous.DelaySequenceMs, current.DelaySequenceMs) {
			add("delaySequenceMs", previous.DelaySequenceMs, current.DelaySequenceMs, nil)
			diff.DelayAssessmentAffected = true
		}
		if previous.InitiationDirection != current.InitiationDirection {
			add("initiationDirection", previous.InitiationDirection, current.InitiationDirection, nil)
			diff.ReviewAttention = append(diff.ReviewAttention, "initiationDirection")
		}
		if previous.PropagationK != current.PropagationK {
			delta := round(current.PropagationK-previous.PropagationK, 6)
			add("propagationK", previous.PropagationK, current.PropagationK, &delta)
			diff.RecalculateAllTargets = true
		}
		if previous.PropagationAlpha != current.PropagationAlpha {
			delta := round(current.PropagationAlpha-previous.PropagationAlpha, 6)
			add("propagationAlpha", previous.PropagationAlpha, current.PropagationAlpha, &delta)
			diff.RecalculateAllTargets = true
		}
	}
	if len(diff.Changes) == 0 {
		return RevisionDiff{}, NewError(CodeEmptyRevision, "修订业务参数与当前版本完全相同")
	}
	if diff.RecalculateAllTargets {
		for _, target := range targets {
			diff.AffectedTargetIDs = append(diff.AffectedTargetIDs, target.TargetID)
		}
		sort.Strings(diff.AffectedTargetIDs)
	}
	return diff, nil
}

func CompareReviewRequirements(reasons []ReviewReason, diff RevisionDiff) []ReviewRequirementComparison {
	changed := map[string]bool{}
	for _, change := range diff.Changes {
		changed[change.Field] = true
	}
	affected := map[string]bool{}
	for _, targetID := range diff.AffectedTargetIDs {
		affected[targetID] = true
	}
	result := make([]ReviewRequirementComparison, 0, len(reasons))
	for _, reason := range reasons {
		comparison := ReviewRequirementComparison{ReasonID: reason.ReasonID}
		if reason.Parameter != "" && changed[reason.Parameter] {
			comparison.MatchingFields = append(comparison.MatchingFields, reason.Parameter)
		}
		if reason.TargetID != "" && affected[reason.TargetID] {
			comparison.MatchingFields = append(comparison.MatchingFields, "affectedTargetIds")
		}
		comparison.Responded = len(comparison.MatchingFields) > 0
		result = append(result, comparison)
	}
	return result
}
