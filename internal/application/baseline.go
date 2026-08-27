package application

import (
	"context"
	"fmt"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

func (s *Service) BaselinePrecheck(ctx context.Context, caseID string) (domain.BaselinePrecheck, error) {
	file, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		if domain.ErrorCodeOf(err) == domain.CodeNotFound {
			return domain.BaselinePrecheck{}, fmt.Errorf("读取基线案卷失败: %v", err)
		}
		return domain.BaselinePrecheck{}, err
	}
	return domain.PrecheckBaseline(caseID, file.Targets), nil
}

func (s *Service) CompleteBaseline(ctx context.Context, caseID string, a Actor, c CompleteBaselineCommand) (BaselineResponse, error) {
	if err := requireRole(a, RoleDesigner); err != nil {
		return BaselineResponse{}, err
	}
	if err := requireMutation(c.ExpectedVersion, c.IdempotencyKey); err != nil {
		return BaselineResponse{}, err
	}
	raw, replay, err := s.store.Mutate(ctx, caseID, c.ExpectedVersion, operation("baseline.complete", caseID), c.IdempotencyKey, a.Role, a.Name, func(f *domain.CaseFile) (store.Mutation, error) {
		if f.Case.State != domain.StateDraft {
			return store.Mutation{}, domain.NewError(domain.CodeState, "仅 draft 案卷可确认基线")
		}
		precheck := domain.PrecheckBaseline(caseID, f.Targets)
		if !precheck.Ready {
			return store.Mutation{}, domain.NewDetailedError(domain.CodeBaselineNotReady, precheck, "基线就绪预检存在阻断问题")
		}
		now := s.now()
		confirmation := domain.NewBaselineConfirmation(precheck, now)
		f.BaselineConfirmation = &confirmation
		f.Case.State = domain.StateBaselineReady
		f.Case.Version++
		f.Case.UpdatedAt = now
		resp := BaselineResponse{CaseResponse: CaseResponse{CaseID: caseID, State: f.Case.State, Version: f.Case.Version}, Precheck: precheck}
		return store.Mutation{EventType: "baseline.completed", Details: map[string]any{"targetCount": len(f.Targets), "typeCounts": confirmation.TypeCounts, "controlTargetId": confirmation.ControlTargetID, "riskCounts": confirmation.RiskCounts}, Response: resp}, nil
	})
	if err != nil {
		return BaselineResponse{}, err
	}
	out, err := decodeResult[BaselineResponse](raw)
	_ = replay
	return out, err
}
