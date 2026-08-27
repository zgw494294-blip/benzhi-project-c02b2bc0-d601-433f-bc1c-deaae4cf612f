package application

import (
	"context"
	"strings"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

func (s *Service) SubmitRevision(ctx context.Context, caseID string, a Actor, c SubmitRevisionCommand, remediation bool) (RevisionResponse, error) {
	if err := requireRole(a, RoleDesigner); err != nil {
		return RevisionResponse{}, err
	}
	if err := requireMutation(c.ExpectedVersion, c.IdempotencyKey); err != nil {
		return RevisionResponse{}, err
	}
	op := "revision.submit"
	event := "revision.submitted"
	if remediation {
		op = "remediation.submit"
		event = "remediation.submitted"
		if strings.TrimSpace(c.RemediationNote) == "" {
			return RevisionResponse{}, domain.NewError(domain.CodeValidation, "整改必须提供 remediationNote")
		}
	}
	raw, replay, err := s.store.Mutate(ctx, caseID, c.ExpectedVersion, operation(op, caseID), c.IdempotencyKey, a.Role, a.Name, func(f *domain.CaseFile) (store.Mutation, error) {
		if !f.Case.State.CanAcceptRevision() {
			return store.Mutation{}, domain.NewError(domain.CodeState, "当前状态 %s 不允许提交设计修订", f.Case.State)
		}
		if remediation && f.Case.State != domain.StateRemediationRequired {
			return store.Mutation{}, domain.NewError(domain.CodeState, "仅 remediation_required 状态可走整改入口")
		}
		if !remediation && f.Case.State == domain.StateRemediationRequired {
			return store.Mutation{}, domain.NewError(domain.CodeState, "存在未关闭阻断项时必须通过整改入口并逐项引用")
		}
		previousRevision := f.CurrentRevision()
		previousAssessment := f.CurrentAssessment()
		if remediation {
			if previousAssessment == nil || !domain.HasBlockers(*previousAssessment) {
				return store.Mutation{}, domain.NewError(domain.CodeState, "当前评估没有可整改的未关闭阻断项")
			}
			if err := validateFindingResolutions(c.FindingResolutions, *previousAssessment); err != nil {
				return store.Mutation{}, err
			}
		}
		now := s.now()
		r := domain.DesignRevision{RevisionID: newID("rev_"), CaseID: caseID, RevisionNumber: len(f.Revisions) + 1, HolePattern: strings.TrimSpace(c.HolePattern), MaxChargePerDelayKg: c.MaxChargePerDelayKg, DelaySequenceMs: append([]int(nil), c.DelaySequenceMs...), InitiationDirection: strings.TrimSpace(c.InitiationDirection), PropagationK: c.PropagationK, PropagationAlpha: c.PropagationAlpha, RemediationNote: strings.TrimSpace(c.RemediationNote), SubmittedAt: now}
		if previousRevision != nil {
			r.ParentRevisionID = previousRevision.RevisionID
		}
		r = domain.NormalizeRevision(r)
		if err := domain.ValidateRevision(r); err != nil {
			return store.Mutation{}, err
		}
		diff, err := domain.BuildRevisionDiff(previousRevision, r, f.Targets)
		if err != nil {
			return store.Mutation{}, err
		}
		r.Diff = diff
		if rejected := domain.LastRejectedReview(f.Reviews); rejected != nil {
			comparisonDiff := diff
			for i := range f.Revisions {
				if f.Revisions[i].RevisionID == rejected.RevisionID {
					if cumulative, buildErr := domain.BuildRevisionDiff(&f.Revisions[i], r, f.Targets); buildErr == nil {
						comparisonDiff = cumulative
					} else {
						comparisonDiff = domain.RevisionDiff{}
					}
					break
				}
			}
			r.ReviewComparison = domain.CompareReviewRequirements(rejected.Reasons, comparisonDiff)
		}
		a := domain.Assess(r, f.Targets, now, newID("assessment_"))
		var transitions []domain.FindingTransition
		if remediation {
			transitions = domain.MergeFindingLifecycle(*previousAssessment, a, c.FindingResolutions, r.RevisionID, now)
			f.FindingHistory = append(f.FindingHistory, transitions...)
		}
		f.Revisions = append(f.Revisions, r)
		f.Assessments = append(f.Assessments, a)
		f.Case.CurrentRevisionID = r.RevisionID
		if domain.HasBlockers(a) {
			f.Case.State = domain.StateRemediationRequired
		} else {
			f.Case.State = domain.StateReviewReady
		}
		f.Case.Version++
		f.Case.UpdatedAt = now
		pending := make([]domain.Finding, 0, len(a.BlockingFindingIDs))
		for _, finding := range a.Findings {
			if finding.Blocking {
				pending = append(pending, finding)
			}
		}
		resp := RevisionResponse{CaseResponse: CaseResponse{CaseID: caseID, State: f.Case.State, Version: f.Case.Version, CurrentRevisionID: r.RevisionID}, Revision: r, Assessment: a, FindingTransitions: transitions, PendingFindings: pending}
		details := map[string]any{"revisionId": r.RevisionID, "parentRevisionId": r.ParentRevisionID, "revisionNumber": r.RevisionNumber, "blockingCount": len(a.BlockingFindingIDs), "diff": diff}
		if remediation {
			closed, stillOpen, introduced := 0, 0, 0
			for _, transition := range transitions {
				switch transition.Status {
				case "closed":
					closed++
				case "still_open", "replaced":
					stillOpen++
				case "new":
					introduced++
				}
			}
			details["closedCount"], details["remainingCount"], details["newCount"] = closed, stillOpen, introduced
		}
		return store.Mutation{EventType: event, Details: details, Response: resp}, nil
	})
	if err != nil {
		return RevisionResponse{}, err
	}
	out, err := decodeResult[RevisionResponse](raw)
	_ = replay
	return out, err
}

func validateFindingResolutions(resolutions []domain.FindingResolution, assessment domain.AssessmentSnapshot) error {
	if len(resolutions) == 0 {
		return domain.NewError(domain.CodeRemediationReference, "整改必须引用至少一个当前未关闭阻断项")
	}
	current := map[string]bool{}
	for _, id := range assessment.BlockingFindingIDs {
		current[id] = true
	}
	seen := map[string]bool{}
	for i, resolution := range resolutions {
		if seen[resolution.FindingID] {
			return domain.NewDetailedError(domain.CodeRemediationReference, map[string]any{"itemIndex": i + 1, "findingId": resolution.FindingID}, "整改请求重复引用同一阻断项")
		}
		if !current[resolution.FindingID] {
			return domain.NewDetailedError(domain.CodeRemediationReference, map[string]any{"itemIndex": i + 1, "findingId": resolution.FindingID}, "整改请求引用了未知或历史阻断项")
		}
		if strings.TrimSpace(resolution.HandlingNote) == "" {
			return domain.NewDetailedError(domain.CodeRemediationReference, map[string]any{"itemIndex": i + 1, "findingId": resolution.FindingID}, "每个阻断项必须提供 handlingNote")
		}
		seen[resolution.FindingID] = true
	}
	return nil
}
