package application

import (
	"context"
	"strings"

	"seed-vigor-gate/internal/domain"
)

func (s *Service) CreateCase(ctx context.Context, command CreateCaseCommand) (*domain.QualificationCase, error) {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.Actor = strings.TrimSpace(command.Actor)
	if err := validateCreateMeta(command.IdempotencyKey, command.Actor); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item, err := domain.CreateCase(domain.NewCase{ID: s.ids.New("case_"), AccessionCode: command.AccessionCode, Source: command.Source, HarvestedAt: command.HarvestedAt, DeclaredSeedCount: command.DeclaredSeedCount, ProtocolCode: command.ProtocolCode, Now: s.clock.Now()})
	if err != nil {
		return nil, err
	}
	created, _, err := s.repository.Create(ctx, command.IdempotencyKey, "case.created", command.Actor, item)
	return created, err
}

func (s *Service) ConfirmSampling(ctx context.Context, caseID string, command ConfirmSamplingCommand) (*domain.QualificationCase, error) {
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.update(ctx, caseID, command.WriteMeta, "sampling.confirmed", func(c *domain.QualificationCase) error {
		return c.ConfirmSampling(domain.ConfirmSamplingInput{ID: s.ids.New("sample_"), SampleUnits: command.SampleUnits, UnitQuotas: command.UnitQuotas, ReplicateCount: command.ReplicateCount, SeedsPerReplicate: command.SeedsPerReplicate, TemperatureMinC: command.TemperatureMinC, TemperatureMaxC: command.TemperatureMaxC, EnvironmentNotes: command.EnvironmentNotes, ConfirmedBy: command.Actor, Now: s.clock.Now()})
	})
}

func (s *Service) update(ctx context.Context, caseID string, meta WriteMeta, action string, mutation Mutation) (*domain.QualificationCase, error) {
	return s.updateWithDetails(ctx, caseID, meta, action, nil, mutation)
}

func (s *Service) updateWithDetails(ctx context.Context, caseID string, meta WriteMeta, action string, details map[string]any, mutation Mutation) (*domain.QualificationCase, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item, _, err := s.repository.Update(ctx, caseID, meta.ExpectedVersion, meta.IdempotencyKey, action, meta.Actor, details, mutation)
	return item, err
}
