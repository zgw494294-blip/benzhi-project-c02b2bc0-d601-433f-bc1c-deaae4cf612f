package application

import (
	"context"
	"strings"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

func (s *Service) Review(ctx context.Context, caseID string, actor Actor, command ReviewCommand) (ReviewResponse, error) {
	if err := requireRole(actor, RoleReviewer); err != nil {
		return ReviewResponse{}, err
	}
	if err := requireMutation(command.ExpectedVersion, command.IdempotencyKey); err != nil {
		return ReviewResponse{}, err
	}
	if command.Decision != "approve" && command.Decision != "reject" {
		return ReviewResponse{}, domain.NewError(domain.CodeValidation, "decision 必须为 approve 或 reject")
	}
	raw, replay, err := s.store.Mutate(ctx, caseID, command.ExpectedVersion, operation("review."+command.Decision, caseID), command.IdempotencyKey, actor.Role, actor.Name, func(file *domain.CaseFile) (store.Mutation, error) {
		if file.Case.State != domain.StateReviewReady {
			return store.Mutation{}, domain.NewError(domain.CodeState, "仅 review_ready 案卷可复核")
		}
		assessment := file.CurrentAssessment()
		if assessment == nil || domain.HasBlockers(*assessment) {
			return store.Mutation{}, domain.NewError(domain.CodeState, "当前修订仍有阻断项")
		}
		now := s.now()
		points := make([]domain.SensorPoint, 0, len(command.SensorPoints))
		for _, point := range command.SensorPoints {
			point.Name = strings.TrimSpace(point.Name)
			point.Location = strings.TrimSpace(point.Location)
			points = append(points, point)
		}
		plan := domain.MonitoringPlan{PlanID: newID("plan_"), CaseID: caseID, RevisionID: file.Case.CurrentRevisionID, AssessmentID: assessment.AssessmentID, SensorPoints: points, SampleRateHz: command.SampleRateHz, TriggerPpvMmPerSec: command.TriggerPpvMmPerSec, EvacuationRule: strings.TrimSpace(command.EvacuationRule), RemainingRisk: strings.TrimSpace(command.RemainingRisk), ReviewDecision: command.Decision, ReviewedBy: actor.Name, ReviewedAt: now}
		if err := domain.ValidateMonitoring(plan, file.Targets); err != nil {
			return store.Mutation{}, err
		}
		validation := domain.BuildMonitoringValidation(plan, file.Targets, *assessment)
		plan.Validation = validation
		if command.Decision == "approve" && !validation.Ready {
			return store.Mutation{}, domain.NewDetailedError(domain.CodeMonitoringInterlock, validation, "监测覆盖或触发阈值联锁校验未通过")
		}
		reasons := normalizeReviewReasons(command.Reasons)
		if command.Decision == "reject" {
			if err := domain.ValidateReviewReasons(reasons, file.Targets); err != nil {
				return store.Mutation{}, err
			}
		}
		comparison, pending, err := validateReReview(command.Decision, command.ReasonResolutions, file)
		if err != nil {
			return store.Mutation{}, err
		}
		round := domain.ReviewRound{Round: len(file.Reviews) + 1, RevisionID: file.Case.CurrentRevisionID, AssessmentID: assessment.AssessmentID, Plan: plan, Decision: command.Decision, Reasons: reasons, ReasonResolutions: append([]domain.ReviewReasonResolution(nil), command.ReasonResolutions...), Comparison: comparison, Validation: validation}
		file.Reviews = append(file.Reviews, round)
		file.MonitoringPlan = &plan
		if command.Decision == "approve" {
			file.Case.State = domain.StateApproved
		} else {
			file.Case.State = domain.StateChangesRequired
		}
		file.Case.Version++
		file.Case.UpdatedAt = now
		response := ReviewResponse{CaseResponse: CaseResponse{CaseID: caseID, State: file.Case.State, Version: file.Case.Version, CurrentRevisionID: file.Case.CurrentRevisionID}, Review: round, PendingReviewReasonIDs: pending}
		eventType := "review.approved"
		if command.Decision == "reject" {
			eventType = "review.rejected"
		}
		details := map[string]any{"planId": plan.PlanID, "round": round.Round, "revisionId": round.RevisionID, "assessmentId": round.AssessmentID, "decision": command.Decision, "reasonCount": len(reasons), "coverageMissingCount": len(validation.UncoveredTargetIDs), "thresholdIssueCount": len(validation.ThresholdIssues)}
		return store.Mutation{EventType: eventType, Details: details, Response: response}, nil
	})
	if err != nil {
		return ReviewResponse{}, err
	}
	out, err := decodeResult[ReviewResponse](raw)
	_ = replay
	return out, err
}

func normalizeReviewReasons(input []ReviewReasonInput) []domain.ReviewReason {
	reasons := make([]domain.ReviewReason, 0, len(input))
	for _, item := range input {
		targetID := item.TargetID
		if targetID == "" {
			targetID = item.RelatedTargetID
		}
		parameter := item.Parameter
		if parameter == "" {
			parameter = item.RelatedParameter
		}
		reason := domain.ReviewReason{ReasonID: newID("reason_"), Category: strings.TrimSpace(item.Category), Description: strings.TrimSpace(item.Description), TargetID: strings.TrimSpace(targetID), Parameter: strings.TrimSpace(parameter), RequiredChange: strings.TrimSpace(item.RequiredChange)}
		reasons = append(reasons, reason)
	}
	return reasons
}

func validateReReview(decision string, resolutions []domain.ReviewReasonResolution, file *domain.CaseFile) ([]domain.ReviewRequirementComparison, []string, error) {
	rejected := domain.LastRejectedReview(file.Reviews)
	if rejected == nil {
		if len(resolutions) > 0 {
			return nil, nil, domain.NewError(domain.CodeReviewUnresolved, "首次复核不存在可确认的历史退回原因")
		}
		return nil, nil, nil
	}
	revision := file.CurrentRevision()
	if revision == nil || revision.ParentRevisionID == "" || revision.RevisionID == rejected.RevisionID {
		return nil, nil, domain.NewError(domain.CodeReviewUnresolved, "退回后尚未形成业务参数有变化的新修订")
	}
	comparison := revision.ReviewComparison
	responded := map[string]bool{}
	for _, item := range comparison {
		responded[item.ReasonID] = item.Responded
	}
	confirmed := map[string]bool{}
	seen := map[string]bool{}
	known := map[string]bool{}
	for _, reason := range rejected.Reasons {
		known[reason.ReasonID] = true
	}
	for i, resolution := range resolutions {
		if !known[resolution.ReasonID] || seen[resolution.ReasonID] {
			return comparison, nil, domain.NewDetailedError(domain.CodeReviewUnresolved, map[string]any{"itemIndex": i + 1, "reasonId": resolution.ReasonID}, "原因处置确认引用未知或重复的原因编号")
		}
		seen[resolution.ReasonID] = true
		confirmed[resolution.ReasonID] = resolution.Confirmed
	}
	pending := []string{}
	for _, reason := range rejected.Reasons {
		if !responded[reason.ReasonID] || !confirmed[reason.ReasonID] {
			pending = append(pending, reason.ReasonID)
		}
	}
	if decision == "approve" && len(pending) > 0 {
		return comparison, pending, domain.NewDetailedError(domain.CodeReviewUnresolved, map[string]any{"reasonIds": pending, "comparison": comparison}, "仍有退回要求未响应或未确认")
	}
	return comparison, pending, nil
}
