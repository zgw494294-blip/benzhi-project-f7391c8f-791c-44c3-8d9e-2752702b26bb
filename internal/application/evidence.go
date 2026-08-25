package application

import (
	"context"

	"seed-vigor-gate/internal/domain"
)

func (s *Service) Freeze(ctx context.Context, caseID string, command FreezeCommand) (*domain.QualificationCase, error) {
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	return s.update(ctx, caseID, command.WriteMeta, "evidence.frozen", func(c *domain.QualificationCase) error {
		return c.Freeze(s.ids.New("bundle_"), command.Actor, s.clock.Now())
	})
}

func (s *Service) IssueCredential(ctx context.Context, caseID string, command IssueCredentialCommand) (*domain.QualificationCase, error) {
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	return s.update(ctx, caseID, command.WriteMeta, "credential.issued", func(c *domain.QualificationCase) error {
		if c.EvidenceBundle == nil {
			return &domain.DomainError{Code: domain.CodeState, Field: "evidenceBundle", Message: "必须先冻结证据包"}
		}
		now := s.clock.Now()
		number := domain.CredentialNumber(c.ID, c.EvidenceBundle.Digest, now)
		return c.IssueCredential(number, command.Actor, now)
	})
}
