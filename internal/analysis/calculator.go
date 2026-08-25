package analysis

import (
	"fmt"
	"math"
	"sort"

	"seed-vigor-gate/internal/domain"
)

type Result struct {
	Metric            domain.Metric
	Findings          []domain.Finding
	FinalObservations []domain.ObservationRevision
	Trajectories      []domain.GerminationTrajectory
}

func Evaluate(c *domain.QualificationCase) Result {
	return evaluateEvidence(c, c.Observations)
}

func EvaluateSupplemental(c *domain.QualificationCase, trial *domain.SupplementalTrial) Result {
	return evaluateEvidence(c, trial.Observations)
}

func evaluateEvidence(c *domain.QualificationCase, observations []domain.ObservationRevision) Result {
	result := Result{Findings: []domain.Finding{}, FinalObservations: []domain.ObservationRevision{}, Trajectories: []domain.GerminationTrajectory{}}
	if c.SamplingPlan == nil {
		result.Findings = append(result.Findings, finding("MISSING_SAMPLING_PLAN", "critical", "尚未确认抽样方案", "case:"+c.ID))
		return result
	}
	protocol := ProtocolFor(c.ProtocolCode)
	formalByDay := latestSubmittedByReplicateAndDay(observations)
	latest := latestSubmittedByReplicate(formalByDay)
	for replicate := 1; replicate <= c.SamplingPlan.ReplicateCount; replicate++ {
		observation, ok := latest[replicate]
		if !ok {
			result.Findings = append(result.Findings, finding("MISSING_OBSERVATION", "critical", fmt.Sprintf("重复组 %d 缺少已提交观测", replicate), fmt.Sprintf("replicate:%d", replicate)))
			continue
		}
		result.FinalObservations = append(result.FinalObservations, observation)
		ref := fmt.Sprintf("observation:%s:r%d", observation.ID, observation.Revision)
		counted := observation.NormalCount + observation.AbnormalCount + observation.UngerminatedCount + observation.ContaminatedCount
		if counted != c.SamplingPlan.SeedsPerReplicate {
			result.Findings = append(result.Findings, finding("COUNT_CONSERVATION", "critical", fmt.Sprintf("重复组 %d 计数不守恒", replicate), ref))
		}
		if observation.DayNo < protocol.MinimumObservationDay {
			result.Findings = append(result.Findings, finding("OBSERVATION_TOO_EARLY", "major", fmt.Sprintf("重复组 %d 尚未达到协议终判日", replicate), ref))
		}
		if observation.TemperatureC < c.SamplingPlan.TemperatureMinC || observation.TemperatureC > c.SamplingPlan.TemperatureMaxC {
			result.Findings = append(result.Findings, finding("TEMPERATURE_OUT_OF_RANGE", "major", fmt.Sprintf("重复组 %d 温度 %.1f℃ 超出 %.1f 至 %.1f℃", replicate, observation.TemperatureC, c.SamplingPlan.TemperatureMinC, c.SamplingPlan.TemperatureMaxC), ref))
		}
		contaminationRate := percent(observation.ContaminatedCount, c.SamplingPlan.SeedsPerReplicate)
		if contaminationRate > protocol.MaximumContaminationRate {
			result.Findings = append(result.Findings, finding("CONTAMINATION_LIMIT", "major", fmt.Sprintf("重复组 %d 污染率 %.2f%% 超限", replicate, contaminationRate), ref))
		}
	}
	analyzeTrajectories(&result, c.SamplingPlan, formalByDay)
	domain.SortObservations(result.FinalObservations)
	calculateMetric(&result, c.SamplingPlan, protocol)
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].RuleCode != result.Findings[j].RuleCode {
			return result.Findings[i].RuleCode < result.Findings[j].RuleCode
		}
		return result.Findings[i].EvidenceRefs[0] < result.Findings[j].EvidenceRefs[0]
	})
	return result
}

func latestSubmittedByReplicate(values []domain.ObservationRevision) map[int]domain.ObservationRevision {
	result := map[int]domain.ObservationRevision{}
	for _, item := range values {
		current, exists := result[item.ReplicateNo]
		if !exists || item.DayNo > current.DayNo || item.DayNo == current.DayNo && item.Revision > current.Revision {
			result[item.ReplicateNo] = item
		}
	}
	return result
}

func latestSubmittedByReplicateAndDay(values []domain.ObservationRevision) []domain.ObservationRevision {
	latest := map[string]domain.ObservationRevision{}
	for _, item := range values {
		if !item.Submitted {
			continue
		}
		key := fmt.Sprintf("%03d/%03d", item.ReplicateNo, item.DayNo)
		if current, ok := latest[key]; !ok || item.Revision > current.Revision {
			latest[key] = item
		}
	}
	result := make([]domain.ObservationRevision, 0, len(latest))
	for _, item := range latest {
		result = append(result, item)
	}
	domain.SortObservations(result)
	return result
}

func analyzeTrajectories(result *Result, plan *domain.SamplingPlan, values []domain.ObservationRevision) {
	byReplicate := map[int][]domain.ObservationRevision{}
	for _, item := range values {
		byReplicate[item.ReplicateNo] = append(byReplicate[item.ReplicateNo], item)
	}
	for replicate := 1; replicate <= plan.ReplicateCount; replicate++ {
		items := byReplicate[replicate]
		plateau := 0
		for index, item := range items {
			germinated := item.NormalCount + item.AbnormalCount
			rate := round2(percent(germinated, plan.SeedsPerReplicate))
			delta := rate
			ref := observationRef(item)
			if index > 0 {
				previous := items[index-1]
				previousGerminated := previous.NormalCount + previous.AbnormalCount
				delta = round2(percent(germinated-previousGerminated, plan.SeedsPerReplicate))
				previousRef := observationRef(previous)
				if germinated < previousGerminated {
					result.Findings = append(result.Findings, domain.Finding{RuleCode: "GERMINATION_CUMULATIVE_DECREASE", Severity: "critical", Message: fmt.Sprintf("重复组 %d 第 %d 日累计发芽数较第 %d 日下降", replicate, item.DayNo, previous.DayNo), EvidenceRefs: []string{previousRef, ref}})
				}
				if item.UngerminatedCount > previous.UngerminatedCount {
					result.Findings = append(result.Findings, domain.Finding{RuleCode: "UNGERMINATED_COUNT_INCREASE", Severity: "major", Message: fmt.Sprintf("重复组 %d 第 %d 日未发芽数较第 %d 日反向增加", replicate, item.DayNo, previous.DayNo), EvidenceRefs: []string{previousRef, ref}})
				}
				if item.DayNo-previous.DayNo > 1 {
					result.Findings = append(result.Findings, domain.Finding{RuleCode: "OBSERVATION_DAY_GAP", Severity: "major", Message: fmt.Sprintf("重复组 %d 在第 %d 日与第 %d 日之间存在观测日缺口", replicate, previous.DayNo, item.DayNo), EvidenceRefs: []string{previousRef, ref}})
				}
				if outOfRange(previous, plan) && outOfRange(item, plan) {
					result.Findings = append(result.Findings, domain.Finding{RuleCode: "CONSECUTIVE_TEMPERATURE_OUT_OF_RANGE", Severity: "critical", Message: fmt.Sprintf("重复组 %d 连续多个观测日温度越界", replicate), EvidenceRefs: []string{previousRef, ref}})
				}
				if germinated == previousGerminated {
					plateau++
				} else {
					plateau = 0
				}
			}
			result.Trajectories = append(result.Trajectories, domain.GerminationTrajectory{ReplicateNo: replicate, DayNo: item.DayNo, ObservationRef: ref, GerminationRate: rate, AdjacentDayDelta: delta, PlateauLength: plateau})
		}
	}
}

func outOfRange(item domain.ObservationRevision, plan *domain.SamplingPlan) bool {
	return item.TemperatureC < plan.TemperatureMinC || item.TemperatureC > plan.TemperatureMaxC
}

func observationRef(item domain.ObservationRevision) string {
	return fmt.Sprintf("observation:%s:r%d", item.ID, item.Revision)
}

func calculateMetric(result *Result, plan *domain.SamplingPlan, protocol Protocol) {
	if len(result.FinalObservations) == 0 {
		result.Metric.CandidateDecision = "insufficient"
		return
	}
	rates := make([]float64, 0, len(result.FinalObservations))
	for _, item := range result.FinalObservations {
		effective := plan.SeedsPerReplicate - item.ContaminatedCount
		result.Metric.EffectiveSampleSize += effective
		result.Metric.NormalTotal += item.NormalCount
		rates = append(rates, percent(item.NormalCount, effective))
	}
	result.Metric.GerminationRate = round2(percent(result.Metric.NormalTotal, result.Metric.EffectiveSampleSize))
	sort.Float64s(rates)
	if len(rates) > 1 {
		result.Metric.ReplicateDifference = round2(rates[len(rates)-1] - rates[0])
	}
	if result.Metric.ReplicateDifference > protocol.MaximumReplicateSpread {
		refs := make([]string, 0, len(result.FinalObservations))
		for _, item := range result.FinalObservations {
			refs = append(refs, fmt.Sprintf("observation:%s:r%d", item.ID, item.Revision))
		}
		result.Findings = append(result.Findings, domain.Finding{RuleCode: "REPLICATE_SPREAD", Severity: "major", Message: fmt.Sprintf("重复组发芽率差 %.2f 个百分点，超过 %.2f", result.Metric.ReplicateDifference, protocol.MaximumReplicateSpread), EvidenceRefs: refs})
	}
	if result.Metric.GerminationRate < protocol.MinimumGerminationRate {
		result.Findings = append(result.Findings, finding("GERMINATION_BELOW_LIMIT", "critical", fmt.Sprintf("发芽率 %.2f%% 低于协议下限 %.2f%%", result.Metric.GerminationRate, protocol.MinimumGerminationRate), "analysis:metric"))
	}
	if len(result.FinalObservations) < plan.ReplicateCount || len(result.Findings) > 0 {
		result.Metric.CandidateDecision = "not_ready"
	} else {
		result.Metric.CandidateDecision = "eligible"
	}
}

func percent(value, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}
func round2(value float64) float64 { return math.Round(value*100) / 100 }
func finding(code, severity, message, ref string) domain.Finding {
	return domain.Finding{RuleCode: code, Severity: severity, Message: message, EvidenceRefs: []string{ref}}
}
