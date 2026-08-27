package application

import (
	"context"
	"fmt"

	"blast-permit/internal/domain"
)

func (s *Service) GetCase(ctx context.Context, caseID string) (*domain.CaseFile, error) {
	file, err := s.store.GetCase(ctx, caseID)
	if domain.ErrorCodeOf(err) == domain.CodeNotFound {
		return nil, fmt.Errorf("查询案卷失败: %v", err)
	}
	return file, err
}
func (s *Service) Audit(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	events, err := s.store.Audit(ctx, caseID)
	if domain.ErrorCodeOf(err) == domain.CodeNotFound {
		return nil, fmt.Errorf("查询审计失败: %v", err)
	}
	return events, err
}
func (s *Service) AuditCount(ctx context.Context, caseID string) (int, error) {
	return s.store.AuditCount(ctx, caseID)
}
