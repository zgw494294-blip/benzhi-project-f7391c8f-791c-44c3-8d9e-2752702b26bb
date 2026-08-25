package domain

import (
	"strings"
	"time"
)

func (c *QualificationCase) ReplaceDetectedDeviations(findings []Finding, idFor func(int, Finding) string) {
	for i := range c.Deviations {
		if c.Deviations[i].Status == "open" {
			c.Deviations[i].Status = "superseded"
		}
	}
	for index, finding := range findings {
		c.Deviations = append(c.Deviations, Deviation{ID: idFor(index, finding), CaseID: c.ID, RuleCode: finding.RuleCode, Severity: finding.Severity, Status: "open", EvidenceRefs: append([]string(nil), finding.EvidenceRefs...)})
	}
}

type ResolveDeviationInput struct {
	DeviationID, Cause, CorrectiveAction, SupplementalTrialID, ResolvedBy string
	StartSupplementalTrial                                                bool
	Now                                                                   time.Time
}

func (c *QualificationCase) ResolveDeviation(input ResolveDeviationInput) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusObserving && c.Status != StatusReturned {
		return invalidState(c.Status, "处置偏差")
	}
	if strings.TrimSpace(input.Cause) == "" || strings.TrimSpace(input.CorrectiveAction) == "" {
		return invalid("deviation", "原因说明和纠正措施均不能为空")
	}
	if strings.TrimSpace(input.ResolvedBy) == "" {
		return invalid("resolvedBy", "处置人不能为空")
	}
	for index := range c.Deviations {
		item := &c.Deviations[index]
		if item.ID != input.DeviationID {
			continue
		}
		if item.Status != "open" {
			return invalid("deviationId", "偏差已关闭、处置中或已被后续分析取代")
		}
		if strings.TrimSpace(input.SupplementalTrialID) != "" && !input.StartSupplementalTrial {
			return invalid("supplementalTrialId", "补充试验编号只能由系统创建，不能直接指定")
		}
		if item.Severity == "critical" && !input.StartSupplementalTrial {
			return invalid("startSupplementalTrial", "严重偏差必须发起补充试验")
		}
		item.Cause = strings.TrimSpace(input.Cause)
		item.CorrectiveAction = strings.TrimSpace(input.CorrectiveAction)
		item.ResolvedBy = strings.TrimSpace(input.ResolvedBy)
		if input.StartSupplementalTrial {
			if input.SupplementalTrialID == "" || c.SamplingPlan == nil {
				return invalid("supplementalTrialId", "系统未能创建有效的补充试验")
			}
			if c.FindSupplementalTrial(input.SupplementalTrialID) != nil {
				return invalid("supplementalTrialId", "补充试验编号重复")
			}
			now := input.Now.UTC()
			item.SupplementalTrialID = input.SupplementalTrialID
			item.Status = "in_progress"
			c.SupplementalTrials = append(c.SupplementalTrials, SupplementalTrial{ID: input.SupplementalTrialID, CaseID: c.ID, DeviationID: item.ID, SamplingPlanID: c.SamplingPlan.ID, InitiatedBy: item.ResolvedBy, InitiatedAt: now, Status: "active", Observations: []ObservationRevision{}, AnalysisSnapshots: []AnalysisSnapshot{}})
			return nil
		}
		item.Status = "resolved"
		item.ResolvedAt = input.Now.UTC()
		return nil
	}
	return &DomainError{Code: CodeNotFound, Field: "deviationId", Message: "偏差不存在"}
}

func (c *QualificationCase) ApplySupplementalAnalysis(trialID string, snapshot AnalysisSnapshot) error {
	trial := c.FindSupplementalTrial(trialID)
	if trial == nil || trial.CaseID != c.ID {
		return invalid("supplementalTrialId", "补充试验不存在或不属于当前批次")
	}
	if trial.Status == "completed" {
		return invalid("supplementalTrialId", "补充试验已经完成")
	}
	covered := make([]bool, c.SamplingPlan.ReplicateCount+1)
	for _, item := range trial.Observations {
		if item.Submitted {
			covered[item.ReplicateNo] = true
		}
	}
	for replicate := 1; replicate <= c.SamplingPlan.ReplicateCount; replicate++ {
		if !covered[replicate] {
			return invalid("supplementalTrialId", "补充试验重复组正式证据尚不齐全")
		}
	}
	trial.Analysis = &snapshot
	trial.AnalysisSnapshots = append(trial.AnalysisSnapshots, snapshot)
	for index := range c.Deviations {
		item := &c.Deviations[index]
		if item.ID != trial.DeviationID || item.SupplementalTrialID != trial.ID {
			continue
		}
		if len(snapshot.Findings) == 0 {
			now := snapshot.CalculatedAt.UTC()
			trial.Status = "completed"
			trial.CompletedAt = &now
			item.Status = "resolved"
			item.ResolvedAt = now
			item.ClosedAnalysisID = snapshot.ID
		} else {
			item.Status = "in_progress"
		}
		return nil
	}
	return invalid("supplementalTrialId", "补充试验未关联当前偏差")
}
