package application

import (
	"context"
	"time"

	"blast-permit/internal/domain"
	"blast-permit/internal/store"
)

func (s *Service) IssuePermit(ctx context.Context, caseID string, a Actor, c IssuePermitCommand) (PermitResponse, error) {
	if err := requireRole(a, RoleSafetyOfficer); err != nil {
		return PermitResponse{}, err
	}
	if err := requireMutation(c.ExpectedVersion, c.IdempotencyKey); err != nil {
		return PermitResponse{}, err
	}
	if c.ValidHours < 1 || c.ValidHours > 24 {
		return PermitResponse{}, domain.NewError(domain.CodeValidation, "validHours 必须为 1 到 24")
	}
	raw, replay, err := s.store.Mutate(ctx, caseID, c.ExpectedVersion, operation("permit.issue", caseID), c.IdempotencyKey, a.Role, a.Name, func(f *domain.CaseFile) (store.Mutation, error) {
		precheck := domain.BuildPermitPrecheck(*f)
		if !precheck.Ready {
			return store.Mutation{}, domain.NewDetailedError(domain.CodePermitPrecheck, precheck, "许可证据预检未通过")
		}
		now := s.now()
		number := "BP-" + now.Format("20060102") + "-" + newID("")[:12]
		permit := domain.NewPermit(number, *f, precheck, a.Name, now, now.Add(time.Duration(c.ValidHours)*time.Hour))
		f.Permit = &permit
		f.Case.State = domain.StateFrozen
		f.Case.Version++
		f.Case.UpdatedAt = now
		resp := PermitResponse{CaseResponse: CaseResponse{CaseID: caseID, State: f.Case.State, Version: f.Case.Version, CurrentRevisionID: f.Case.CurrentRevisionID}, Permit: permit}
		return store.Mutation{EventType: "permit.issued", Details: map[string]any{"permitNumber": number, "evidenceDigest": permit.EvidenceDigest, "components": permit.FrozenComponents, "frozenAuditSequence": permit.FrozenAuditSequence}, Response: resp}, nil
	})
	if err != nil {
		return PermitResponse{}, err
	}
	out, err := decodeResult[PermitResponse](raw)
	_ = replay
	return out, err
}

func (s *Service) PermitPrecheck(ctx context.Context, caseID string) (domain.PermitPrecheck, error) {
	file, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return domain.PermitPrecheck{}, err
	}
	return domain.BuildPermitPrecheck(*file), nil
}

func (s *Service) VerifyPermit(ctx context.Context, number string) (VerificationResponse, error) {
	p, err := s.store.GetPermit(ctx, number)
	if err != nil {
		return VerificationResponse{}, err
	}
	file, err := s.store.GetCase(ctx, p.CaseID)
	if err != nil {
		return VerificationResponse{}, err
	}
	response := VerificationResponse{PermitNumber: p.PermitNumber, CaseID: p.CaseID, EvidenceDigest: p.EvidenceDigest, Status: "valid"}
	if !domain.VerifyPermit(*p) || domain.TotalEvidenceDigest(p.CaseID, p.FrozenComponents) != p.EvidenceDigest {
		response.Status = "permit_digest_mismatch"
		response.FaultComponents = []string{"permit"}
		return response, nil
	}
	current := domain.EvidenceComponents(*file)
	faults := componentMismatches(p.FrozenComponents, current)
	if file.Case.State != domain.StateFrozen || file.Case.CurrentRevisionID != p.FrozenRevisionID {
		faults = append(faults, "case")
	}
	if len(faults) > 0 {
		response.Status = "frozen_evidence_mismatch"
		response.FaultComponents = faults
		return response, nil
	}
	if !file.AuditState.Continuous || file.AuditState.Digests[p.FrozenAuditSequence] != p.FrozenAuditHeadDigest || file.AuditState.Sequence != p.FrozenAuditSequence+1 || file.AuditState.EventTypes[p.FrozenAuditSequence+1] != "permit.issued" {
		response.Status = "audit_chain_broken"
		response.FaultComponents = []string{"audit"}
		return response, nil
	}
	if !p.ValidUntil.After(s.now()) {
		response.Status = "expired"
		return response, nil
	}
	response.Valid = true
	return response, nil
}

func componentMismatches(frozen, current []domain.EvidenceComponent) []string {
	currentByName := map[string]domain.EvidenceComponent{}
	for _, component := range current {
		currentByName[component.Name] = component
	}
	faults := []string{}
	for _, component := range frozen {
		actual, ok := currentByName[component.Name]
		if !ok || actual.Status != component.Status || actual.Digest != component.Digest {
			faults = append(faults, component.Name)
		}
	}
	return faults
}
