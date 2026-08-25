package domain

import (
	"strings"
	"time"
)

func (c *QualificationCase) ApplyAnalysis(snapshot AnalysisSnapshot) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusObserving && c.Status != StatusReturned {
		return invalidState(c.Status, "执行活力分析")
	}
	c.Analysis = &snapshot
	c.AnalysisSnapshots = append(c.AnalysisSnapshots, snapshot)
	if len(snapshot.Findings) == 0 && len(c.OpenDeviations()) == 0 {
		c.Status = StatusPendingReview
	} else {
		c.Status = StatusObserving
	}
	return nil
}

type ReviewInput struct {
	ID, Decision, Reason, Reviewer string
	Now                            time.Time
}

func (c *QualificationCase) Review(input ReviewInput) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusPendingReview {
		return invalidState(c.Status, "质量复核")
	}
	decision := strings.ToLower(strings.TrimSpace(input.Decision))
	if decision != "approve" && decision != "return" {
		return invalid("decision", "复核决定只能为 approve 或 return")
	}
	if strings.TrimSpace(input.Reviewer) == "" {
		return invalid("reviewer", "复核员不能为空")
	}
	if decision == "return" && strings.TrimSpace(input.Reason) == "" {
		return invalid("reason", "退回必须填写理由")
	}
	if c.Analysis == nil {
		return invalid("analysis", "缺少最新分析结果")
	}
	if len(c.OpenDeviations()) != 0 || len(c.Analysis.Findings) != 0 {
		return invalid("deviations", "存在未闭环偏差，不能批准")
	}
	c.Reviews = append(c.Reviews, ReviewDecision{ID: input.ID, Decision: decision, Reason: strings.TrimSpace(input.Reason), Reviewer: strings.TrimSpace(input.Reviewer), CaseVersion: c.Version, AnalysisID: c.Analysis.ID, DecidedAt: input.Now.UTC()})
	if decision == "approve" {
		c.Status = StatusApproved
	} else {
		c.Status = StatusReturned
	}
	return nil
}
