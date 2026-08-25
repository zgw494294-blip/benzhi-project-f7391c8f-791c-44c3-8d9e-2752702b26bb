package application

import (
	"context"
	"crypto/sha256"
	"fmt"

	"seed-vigor-gate/internal/analysis"
	"seed-vigor-gate/internal/domain"
)

func (s *Service) RecordObservation(ctx context.Context, caseID string, command RecordObservationCommand) (*ObservationBatchResult, error) {
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	rows := command.Rows
	if len(rows) == 0 {
		rows = []ObservationRowCommand{{ReplicateNo: command.ReplicateNo, DayNo: command.DayNo, NormalCount: command.NormalCount, AbnormalCount: command.AbnormalCount, UngerminatedCount: command.UngerminatedCount, ContaminatedCount: command.ContaminatedCount, TemperatureC: command.TemperatureC, Submitted: command.Submitted, SupplementalTrialID: command.SupplementalTrialID}}
	}
	if len(rows) > 20 {
		return nil, fieldError("rows", "观测矩阵每批最多提交 20 行")
	}
	inputs := make([]domain.RecordObservationInput, len(rows))
	now := s.clock.Now()
	for index, row := range rows {
		trialID := row.SupplementalTrialID
		if trialID == "" {
			trialID = command.SupplementalTrialID
		}
		inputs[index] = domain.RecordObservationInput{ID: s.ids.New("obs_"), ReplicateNo: row.ReplicateNo, DayNo: row.DayNo, NormalCount: row.NormalCount, AbnormalCount: row.AbnormalCount, UngerminatedCount: row.UngerminatedCount, ContaminatedCount: row.ContaminatedCount, TemperatureC: row.TemperatureC, RecordedBy: command.Actor, Submitted: row.Submitted, SupplementalTrialID: trialID, OperationKey: observationBatchID(command.IdempotencyKey), Now: now}
	}
	details := map[string]any{"rowCount": len(rows)}
	item, _, err := s.repository.Update(ctx, caseID, command.ExpectedVersion, command.IdempotencyKey, "observations.batch_recorded", command.Actor, details, func(c *domain.QualificationCase) error {
		revisions, err := c.RecordObservations(inputs)
		if err != nil {
			return err
		}
		draft, submitted := 0, 0
		for _, revision := range revisions {
			if revision.Submitted {
				submitted++
			} else {
				draft++
			}
		}
		details["draftCount"] = draft
		details["submittedCount"] = submitted
		details["supplementalTrialId"] = inputs[0].SupplementalTrialID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return batchResult(item, observationBatchID(command.IdempotencyKey)), nil
}

func batchResult(c *domain.QualificationCase, operationKey string) *ObservationBatchResult {
	values := make([]domain.ObservationRevision, 0)
	for _, item := range c.Observations {
		if item.BatchID == operationKey {
			values = append(values, item)
		}
	}
	for _, trial := range c.SupplementalTrials {
		for _, item := range trial.Observations {
			if item.BatchID == operationKey {
				values = append(values, item)
			}
		}
	}
	domain.SortObservations(values)
	result := &ObservationBatchResult{AcceptedRows: len(values), Version: c.Version, Status: c.Status, Revisions: make([]ObservationRevisionResult, 0, len(values))}
	for _, item := range values {
		if item.Submitted {
			result.SubmittedCount++
		} else {
			result.DraftCount++
		}
		result.Revisions = append(result.Revisions, ObservationRevisionResult{ReplicateNo: item.ReplicateNo, DayNo: item.DayNo, Revision: item.Revision, ObservationID: item.ID, SupplementalTrialID: item.SupplementalTrialID})
	}
	return result
}

func observationBatchID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("batch_%x", sum[:12])
}

func (s *Service) Analyze(ctx context.Context, caseID string, command AnalyzeCommand) (*domain.QualificationCase, error) {
	if err := validateMeta(command.WriteMeta); err != nil {
		return nil, err
	}
	action := "analysis.completed"
	if command.SupplementalTrialID != "" {
		action = "supplemental_trial.analysis_completed"
	}
	details := map[string]any{}
	if command.SupplementalTrialID != "" {
		details["supplementalTrialId"] = command.SupplementalTrialID
	}
	return s.updateWithDetails(ctx, caseID, command.WriteMeta, action, details, func(c *domain.QualificationCase) error {
		if command.SupplementalTrialID != "" {
			trial := c.FindSupplementalTrial(command.SupplementalTrialID)
			if trial == nil || trial.CaseID != c.ID {
				return &domain.DomainError{Code: domain.CodeValidation, Field: "supplementalTrialId", Message: "补充试验不存在或不属于当前批次"}
			}
			result := analysis.EvaluateSupplemental(c, trial)
			sequence := len(trial.AnalysisSnapshots) + 1
			snapshot := domain.AnalysisSnapshot{ID: s.ids.New("analysis_"), CaseID: c.ID, Sequence: sequence, CalculatedAt: s.clock.Now(), Metric: result.Metric, Findings: result.Findings, Trajectories: result.Trajectories, SupplementalTrialID: trial.ID}
			return c.ApplySupplementalAnalysis(trial.ID, snapshot)
		}
		result := analysis.Evaluate(c)
		result.Findings = findingsAfterSupplementalClosure(c, result.Findings)
		if len(result.Findings) == 0 && len(result.FinalObservations) == c.SamplingPlan.ReplicateCount {
			result.Metric.CandidateDecision = "eligible"
		}
		sequence := len(c.AnalysisSnapshots) + 1
		snapshot := domain.AnalysisSnapshot{ID: s.ids.New("analysis_"), CaseID: c.ID, Sequence: sequence, CalculatedAt: s.clock.Now(), Metric: result.Metric, Findings: result.Findings, Trajectories: result.Trajectories}
		c.ReplaceDetectedDeviations(result.Findings, func(index int, finding domain.Finding) string { return s.ids.New(fmt.Sprintf("dev_%02d_", index+1)) })
		return c.ApplyAnalysis(snapshot)
	})
}

func findingsAfterSupplementalClosure(c *domain.QualificationCase, findings []domain.Finding) []domain.Finding {
	closedRules := map[string]bool{}
	activeRules := map[string]bool{}
	for _, deviation := range c.Deviations {
		if deviation.Status == "resolved" && deviation.SupplementalTrialID != "" && deviation.ClosedAnalysisID != "" {
			closedRules[deviation.RuleCode] = true
		}
		if deviation.Status == "open" || deviation.Status == "in_progress" {
			activeRules[deviation.RuleCode] = true
		}
	}
	result := make([]domain.Finding, 0, len(findings))
	for _, finding := range findings {
		if !closedRules[finding.RuleCode] || activeRules[finding.RuleCode] {
			result = append(result, finding)
		}
	}
	return result
}
