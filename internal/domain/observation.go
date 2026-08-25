package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type RecordObservationInput struct {
	ID                                                                                   string
	ReplicateNo, DayNo, NormalCount, AbnormalCount, UngerminatedCount, ContaminatedCount int
	TemperatureC                                                                         float64
	RecordedBy                                                                           string
	Submitted                                                                            bool
	SupplementalTrialID, OperationKey                                                    string
	Now                                                                                  time.Time
}

func (c *QualificationCase) RecordObservation(input RecordObservationInput) (*ObservationRevision, error) {
	rows, err := c.RecordObservations([]RecordObservationInput{input})
	if err != nil {
		if issue, ok := err.(*DomainError); ok && strings.HasPrefix(issue.Field, "rows[0].") {
			copy := *issue
			copy.Field = strings.TrimPrefix(copy.Field, "rows[0].")
			return nil, &copy
		}
		return nil, err
	}
	return &rows[0], nil
}

// RecordObservations 先验证全部矩阵行，全部通过后才一次追加。
func (c *QualificationCase) RecordObservations(inputs []RecordObservationInput) ([]ObservationRevision, error) {
	if err := c.EnsureMutable(); err != nil {
		return nil, err
	}
	if c.Status != StatusSamplingConfirmed && c.Status != StatusObserving && c.Status != StatusReturned && c.Status != StatusPendingReview {
		return nil, invalidState(c.Status, "录入观测")
	}
	if c.SamplingPlan == nil {
		return nil, invalid("samplingPlan", "必须先确认抽样方案")
	}
	if len(inputs) < 1 || len(inputs) > 20 {
		return nil, invalid("rows", "观测矩阵每批须包含 1 至 20 行")
	}
	trialID := strings.TrimSpace(inputs[0].SupplementalTrialID)
	var trial *SupplementalTrial
	if trialID != "" {
		trial = c.findSupplementalTrial(trialID)
		if trial == nil {
			return nil, invalid("supplementalTrialId", "补充试验不存在或不属于当前批次")
		}
		if trial.Status == "completed" {
			return nil, invalid("supplementalTrialId", "已完成的补充试验不能继续写入")
		}
	}
	seen := map[string]bool{}
	ordered := append([]RecordObservationInput(nil), inputs...)
	existing := c.Observations
	if trial != nil {
		existing = trial.Observations
	}
	result := make([]ObservationRevision, 0, len(ordered))
	for index, input := range ordered {
		field := fmt.Sprintf("rows[%d]", index)
		if strings.TrimSpace(input.SupplementalTrialID) != trialID {
			return nil, invalid(field+".supplementalTrialId", "同一批观测必须属于同一试验")
		}
		if input.ID == "" {
			return nil, invalid(field+".id", "观测 ID 不能为空")
		}
		if input.ReplicateNo < 1 || input.ReplicateNo > c.SamplingPlan.ReplicateCount {
			return nil, invalid(field+".replicateNo", "重复组编号超出抽样方案")
		}
		if input.DayNo < 1 || input.DayNo > 60 {
			return nil, invalid(field+".dayNo", "观测日须在 1 至 60 之间")
		}
		key := fmt.Sprintf("%03d/%03d", input.ReplicateNo, input.DayNo)
		if seen[key] {
			return nil, invalid(field+".dayNo", "批内重复组与观测日组合不能重复")
		}
		seen[key] = true
		for _, count := range []int{input.NormalCount, input.AbnormalCount, input.UngerminatedCount, input.ContaminatedCount} {
			if count < 0 {
				return nil, invalid(field+".counts", "观测计数不能为负数")
			}
		}
		if input.NormalCount+input.AbnormalCount+input.UngerminatedCount+input.ContaminatedCount != c.SamplingPlan.SeedsPerReplicate {
			return nil, invalid(field+".counts", fmt.Sprintf("四类计数之和必须等于每组计划粒数 %d", c.SamplingPlan.SeedsPerReplicate))
		}
		if input.TemperatureC < -20 || input.TemperatureC > 70 {
			return nil, invalid(field+".temperatureC", "温度读数超出可接受录入范围")
		}
		if strings.TrimSpace(input.RecordedBy) == "" {
			return nil, invalid(field+".recordedBy", "记录人不能为空")
		}
		revision := 1
		for _, item := range existing {
			if item.ReplicateNo == input.ReplicateNo && item.DayNo == input.DayNo && item.Revision >= revision {
				revision = item.Revision + 1
			}
		}
		result = append(result, ObservationRevision{ID: input.ID, CaseID: c.ID, ReplicateNo: input.ReplicateNo, DayNo: input.DayNo, Revision: revision, NormalCount: input.NormalCount, AbnormalCount: input.AbnormalCount, UngerminatedCount: input.UngerminatedCount, ContaminatedCount: input.ContaminatedCount, TemperatureC: input.TemperatureC, RecordedBy: strings.TrimSpace(input.RecordedBy), RecordedAt: input.Now.UTC(), Submitted: input.Submitted, SupplementalTrialID: trialID, BatchID: input.OperationKey})
	}
	SortObservations(result)
	if trial == nil {
		c.Observations = append(c.Observations, result...)
		c.Analysis = nil
	} else {
		trial.Observations = append(trial.Observations, result...)
		trial.Analysis = nil
	}
	if c.Status == StatusSamplingConfirmed || c.Status == StatusReturned || c.Status == StatusPendingReview {
		c.Status = StatusObserving
	}
	return result, nil
}

func (c *QualificationCase) FindSupplementalTrial(id string) *SupplementalTrial {
	for index := range c.SupplementalTrials {
		if c.SupplementalTrials[index].ID == id {
			return &c.SupplementalTrials[index]
		}
	}
	return nil
}

func (c *QualificationCase) findSupplementalTrial(id string) *SupplementalTrial {
	return c.FindSupplementalTrial(id)
}

func SortObservations(values []ObservationRevision) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].SupplementalTrialID != values[j].SupplementalTrialID {
			return values[i].SupplementalTrialID < values[j].SupplementalTrialID
		}
		if values[i].ReplicateNo != values[j].ReplicateNo {
			return values[i].ReplicateNo < values[j].ReplicateNo
		}
		if values[i].DayNo != values[j].DayNo {
			return values[i].DayNo < values[j].DayNo
		}
		return values[i].Revision < values[j].Revision
	})
}
