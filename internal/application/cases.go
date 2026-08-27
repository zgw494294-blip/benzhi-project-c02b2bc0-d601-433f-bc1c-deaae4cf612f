package application

import (
	"context"
	"strings"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

func (s *Service) CreateCase(ctx context.Context, a Actor, c CreateCaseCommand) (CaseResponse, error) {
	if err := requireRole(a, RoleDesigner); err != nil {
		return CaseResponse{}, err
	}
	if err := requireKey(c.IdempotencyKey); err != nil {
		return CaseResponse{}, err
	}
	if err := domain.ValidateNewCase(c.SiteName, c.WorkZone); err != nil {
		return CaseResponse{}, err
	}
	now := s.now()
	id := newID("case_")
	f := &domain.CaseFile{Case: domain.BlastCase{CaseID: id, SiteName: strings.TrimSpace(c.SiteName), WorkZone: strings.TrimSpace(c.WorkZone), State: domain.StateDraft, Version: 1, CreatedAt: now, UpdatedAt: now}}
	resp := CaseResponse{CaseID: id, State: f.Case.State, Version: 1}
	raw, replay, err := s.store.Create(ctx, f, "case.create", c.IdempotencyKey, a.Role, a.Name, resp)
	if err != nil {
		return CaseResponse{}, err
	}
	out, err := decodeResult[CaseResponse](raw)
	_ = replay
	return out, err
}

func (s *Service) AddTargets(ctx context.Context, caseID string, a Actor, c AddTargetsCommand) (AddTargetsResponse, error) {
	if err := requireRole(a, RoleDesigner); err != nil {
		return AddTargetsResponse{}, err
	}
	if err := requireMutation(c.ExpectedVersion, c.IdempotencyKey); err != nil {
		return AddTargetsResponse{}, err
	}
	if len(c.Targets) == 0 {
		return AddTargetsResponse{}, domain.NewError(domain.CodeValidation, "targets 不能为空")
	}
	raw, replay, err := s.store.Mutate(ctx, caseID, c.ExpectedVersion, operation("targets.add", caseID), c.IdempotencyKey, a.Role, a.Name, func(f *domain.CaseFile) (store.Mutation, error) {
		if f.Case.State != domain.StateDraft {
			return store.Mutation{}, domain.NewError(domain.CodeState, "仅 draft 案卷可登记保护对象")
		}
		added := make([]string, 0, len(c.Targets))
		batch := make([]domain.ProtectedTarget, 0, len(c.Targets))
		for _, in := range c.Targets {
			t := domain.ProtectedTarget{TargetID: newID("target_"), CaseID: caseID, TargetType: in.TargetType, Name: strings.TrimSpace(in.Name), DistanceMeters: in.DistanceMeters, AllowedPpvMmPerSec: in.AllowedPpvMmPerSec, BaselinePpvMmPerSec: in.BaselinePpvMmPerSec, MeasurementNote: strings.TrimSpace(in.MeasurementNote)}
			batch = append(batch, t)
		}
		if issues := domain.ValidateTargetBatch(batch, f.Targets); len(issues) > 0 {
			code := domain.CodeValidation
			for _, issue := range issues {
				if issue.Code == "duplicate_existing_target" || issue.Code == "duplicate_batch_target" {
					code = domain.CodeTargetConflict
					break
				}
			}
			return store.Mutation{}, domain.NewDetailedError(code, map[string]any{"issues": issues}, "保护对象批次校验失败")
		}
		for _, target := range batch {
			f.Targets = append(f.Targets, target)
			added = append(added, target.TargetID)
		}
		f.Case.Version++
		f.Case.UpdatedAt = s.now()
		resp := AddTargetsResponse{CaseResponse: CaseResponse{CaseID: caseID, State: f.Case.State, Version: f.Case.Version}, Targets: batch}
		return store.Mutation{EventType: "baseline.targets_added", Details: map[string]any{"targetIds": added}, Response: resp}, nil
	})
	if err != nil {
		return AddTargetsResponse{}, err
	}
	out, err := decodeResult[AddTargetsResponse](raw)
	_ = replay
	return out, err
}
