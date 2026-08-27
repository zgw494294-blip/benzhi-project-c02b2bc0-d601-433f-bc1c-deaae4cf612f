package application

import (
	"context"

	"blast-permit/internal/domain"
)

func (s *Service) GetCase(ctx context.Context, caseID string) (*domain.CaseFile, error) {
	return s.store.GetCase(ctx, caseID)
}
func (s *Service) Audit(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	return s.store.Audit(ctx, caseID)
}
func (s *Service) AuditCount(ctx context.Context, caseID string) (int, error) {
	return s.store.AuditCount(ctx, caseID)
}
