package application

import (
	"context"

	"seed-vigor-gate/internal/domain"
)

func (s *Service) ResolveDeviation(ctx context.Context, caseID, deviationID string, command ResolveDeviationCommand) (*domain.QualificationCase, error) {
	ctx = context.WithoutCancel(ctx)
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	if command.StartSupplementalTrial && command.SupplementalTrialID != "" {
		return nil, fieldError("supplementalTrialId", "发起补充试验时编号必须由系统生成")
	}
	action := "deviation.resolved"
	trialID := command.SupplementalTrialID
	if command.StartSupplementalTrial {
		action = "supplemental_trial.initiated"
		trialID = s.ids.New("trial_")
	}
	details := map[string]any{}
	if trialID != "" {
		details["supplementalTrialId"] = trialID
	}
	return s.updateWithDetails(ctx, caseID, command.WriteMeta, action, details, func(c *domain.QualificationCase) error {
		return c.ResolveDeviation(domain.ResolveDeviationInput{DeviationID: deviationID, Cause: command.Cause, CorrectiveAction: command.CorrectiveAction, SupplementalTrialID: trialID, StartSupplementalTrial: command.StartSupplementalTrial, ResolvedBy: command.Actor, Now: s.clock.Now()})
	})
}

func (s *Service) Review(ctx context.Context, caseID string, command ReviewCommand) (*domain.QualificationCase, error) {
	ctx = context.WithoutCancel(ctx)
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	action := "review." + command.Decision
	return s.update(ctx, caseID, command.WriteMeta, action, func(c *domain.QualificationCase) error {
		return c.Review(domain.ReviewInput{ID: s.ids.New("review_"), Decision: command.Decision, Reason: command.Reason, Reviewer: command.Actor, Now: s.clock.Now()})
	})
}
