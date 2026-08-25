package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ConfirmSamplingInput struct {
	ID                                string
	SampleUnits                       []string
	UnitQuotas                        []SampleUnitQuota
	ReplicateCount, SeedsPerReplicate int
	TemperatureMinC, TemperatureMaxC  float64
	EnvironmentNotes, ConfirmedBy     string
	Now                               time.Time
}

func (c *QualificationCase) ConfirmSampling(input ConfirmSamplingInput) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusDraft {
		return invalidState(c.Status, "确认抽样")
	}
	if input.ID == "" {
		return invalid("samplingPlan.id", "抽样方案 ID 不能为空")
	}
	if input.ReplicateCount < 2 || input.ReplicateCount > 20 {
		return invalid("replicateCount", "重复组须在 2 至 20 组之间")
	}
	if input.SeedsPerReplicate < 25 || input.SeedsPerReplicate > 1000 {
		return invalid("seedsPerReplicate", "每组计划粒数须在 25 至 1000 之间")
	}
	total := input.ReplicateCount * input.SeedsPerReplicate
	if total > c.DeclaredSeedCount {
		return invalid("seedsPerReplicate", "计划抽样总数不能超过声明数量")
	}
	if total < 100 {
		return invalid("replicateCount", "代表性抽样总数不得少于 100 粒")
	}
	quotas, units, err := normalizeUnitQuotas(input, total)
	if err != nil {
		return err
	}
	if input.TemperatureMinC < 0 || input.TemperatureMaxC > 50 || input.TemperatureMinC >= input.TemperatureMaxC {
		return invalid("temperatureMinC", "温度范围须在 0 至 50℃ 内且下限小于上限")
	}
	if strings.TrimSpace(input.ConfirmedBy) == "" {
		return invalid("confirmedBy", "确认人不能为空")
	}
	now := input.Now.UTC()
	c.SamplingPlan = &SamplingPlan{ID: input.ID, CaseID: c.ID, Revision: 1, SampleUnits: units, UnitQuotas: quotas, ReplicateCount: input.ReplicateCount, SeedsPerReplicate: input.SeedsPerReplicate, TemperatureMinC: input.TemperatureMinC, TemperatureMaxC: input.TemperatureMaxC, EnvironmentNotes: strings.TrimSpace(input.EnvironmentNotes), TotalSampled: total, SamplingRatio: math.Round(float64(total)*10000/float64(c.DeclaredSeedCount)) / 100, RemainingReserve: c.DeclaredSeedCount - total, ConfirmedBy: strings.TrimSpace(input.ConfirmedBy), ConfirmedAt: &now}
	c.Status = StatusSamplingConfirmed
	c.Analysis = nil
	return nil
}

func normalizeUnitQuotas(input ConfirmSamplingInput, expectedTotal int) ([]SampleUnitQuota, []string, error) {
	quotas := append([]SampleUnitQuota(nil), input.UnitQuotas...)
	// 兼容既有公开请求：只有 sampleUnits 时按重复组顺序形成等额配额。
	if len(quotas) == 0 {
		if len(input.SampleUnits) != input.ReplicateCount {
			return nil, nil, invalid("unitQuotas", "必须为每个样品单元提交配额和重复组分配")
		}
		for index, name := range input.SampleUnits {
			quotas = append(quotas, SampleUnitQuota{UnitName: name, PlannedCount: input.SeedsPerReplicate, ReplicateNos: []int{index + 1}})
		}
	}
	seenNames := map[string]bool{}
	assigned := make([]bool, input.ReplicateCount+1)
	total := 0
	for index := range quotas {
		quota := &quotas[index]
		quota.UnitName = strings.TrimSpace(quota.UnitName)
		field := "unitQuotas[" + strconv.Itoa(index) + "]"
		if quota.UnitName == "" || seenNames[quota.UnitName] {
			return nil, nil, invalid(field+".unitName", "样品单元名称必须非空且唯一")
		}
		seenNames[quota.UnitName] = true
		if quota.PlannedCount <= 0 {
			return nil, nil, invalid(field+".plannedCount", "样品单元配额必须为正数")
		}
		if len(quota.ReplicateNos) == 0 {
			return nil, nil, invalid(field+".replicateNos", "每个样品单元必须分配至少一个重复组")
		}
		sort.Ints(quota.ReplicateNos)
		for _, replicate := range quota.ReplicateNos {
			if replicate < 1 || replicate > input.ReplicateCount {
				return nil, nil, invalid(field+".replicateNos", "重复组编号超出抽样方案")
			}
			if assigned[replicate] {
				return nil, nil, invalid(field+".replicateNos", "同一重复组不能分配给多个样品单元")
			}
			assigned[replicate] = true
		}
		if quota.PlannedCount != len(quota.ReplicateNos)*input.SeedsPerReplicate {
			return nil, nil, invalid(field+".plannedCount", "样品单元配额必须等于所分配重复组的计划粒数")
		}
		total += quota.PlannedCount
	}
	for replicate := 1; replicate <= input.ReplicateCount; replicate++ {
		if !assigned[replicate] {
			return nil, nil, invalid("unitQuotas", "重复组必须恰好归属一个样品单元")
		}
	}
	if total != expectedTotal {
		return nil, nil, invalid("unitQuotas", "样品单元配额之和必须等于重复组数与每组粒数的乘积")
	}
	sort.Slice(quotas, func(i, j int) bool { return quotas[i].UnitName < quotas[j].UnitName })
	units := make([]string, len(quotas))
	for index := range quotas {
		units[index] = quotas[index].UnitName
	}
	return quotas, units, nil
}
