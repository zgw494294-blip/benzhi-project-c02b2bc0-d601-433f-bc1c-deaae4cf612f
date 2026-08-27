package application

import (
	"context"

	"blast-permit/internal/domain"
)

func (s *Service) GetCase(ctx context.Context, caseID string) (*domain.CaseFile, error) {
	s.caseLoadMu.Lock()
	if current := s.inFlight; current != nil {
		s.caseLoadMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-current.done:
			return current.file, current.err
		}
	}
	current := &caseLoad{done: make(chan struct{})}
	s.inFlight = current
	s.caseLoadMu.Unlock()

	current.file, current.err = s.caseRepository.GetCase(ctx, caseID)
	s.caseLoadMu.Lock()
	s.inFlight = nil
	close(current.done)
	s.caseLoadMu.Unlock()
	return current.file, current.err
}
func (s *Service) Audit(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	return s.store.Audit(ctx, caseID)
}
func (s *Service) AuditCount(ctx context.Context, caseID string) (int, error) {
	return s.store.AuditCount(ctx, caseID)
}
