package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var accessionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,39}$`)
var protocolPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{2,31}$`)

type NewCase struct {
	ID, AccessionCode, Source, HarvestedAt, ProtocolCode string
	DeclaredSeedCount                                    int
	Now                                                  time.Time
}

func CreateCase(input NewCase) (*QualificationCase, error) {
	input.AccessionCode = strings.TrimSpace(input.AccessionCode)
	input.Source = strings.TrimSpace(input.Source)
	input.ProtocolCode = strings.ToUpper(strings.TrimSpace(input.ProtocolCode))
	if input.ID == "" {
		return nil, invalid("id", "批次 ID 不能为空")
	}
	if !accessionPattern.MatchString(input.AccessionCode) {
		return nil, invalid("accessionCode", "种质编号须为 3 至 40 位字母、数字、点、下划线或连字符")
	}
	if len([]rune(input.Source)) < 2 || len([]rune(input.Source)) > 200 {
		return nil, invalid("source", "来源长度须为 2 至 200 个字符")
	}
	harvest, err := time.Parse("2006-01-02", input.HarvestedAt)
	if err != nil {
		return nil, invalid("harvestedAt", "收获日期须使用 YYYY-MM-DD")
	}
	if harvest.After(input.Now.Add(24 * time.Hour)) {
		return nil, invalid("harvestedAt", "收获日期不能晚于当前日期")
	}
	if input.DeclaredSeedCount < 200 || input.DeclaredSeedCount > 10000000 {
		return nil, invalid("declaredSeedCount", "声明数量须在 200 至 10000000 之间")
	}
	if !protocolPattern.MatchString(input.ProtocolCode) {
		return nil, invalid("protocolCode", "鉴定协议须为大写技术标识")
	}
	now := input.Now.UTC()
	return &QualificationCase{ID: input.ID, AccessionCode: input.AccessionCode, Source: input.Source, HarvestedAt: input.HarvestedAt, DeclaredSeedCount: input.DeclaredSeedCount, ProtocolCode: input.ProtocolCode, Status: StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now, Observations: []ObservationRevision{}, SupplementalTrials: []SupplementalTrial{}, AnalysisSnapshots: []AnalysisSnapshot{}, Deviations: []Deviation{}, Reviews: []ReviewDecision{}}, nil
}

func (c *QualificationCase) EnsureMutable() error {
	if c.Status == StatusFrozen || c.Status == StatusCredentialIssued || c.EvidenceBundle != nil {
		return frozenError()
	}
	return nil
}

func (c *QualificationCase) Validate() error {
	if c.ID == "" || c.Version < 1 || c.CreatedAt.IsZero() {
		return invalid("case", "批次基础元数据不完整")
	}
	if c.Status == StatusFrozen && c.EvidenceBundle == nil {
		return invalid("evidenceBundle", "冻结状态必须包含证据包")
	}
	if c.Status == StatusCredentialIssued && (c.EvidenceBundle == nil || c.Credential == nil) {
		return invalid("credential", "凭据签发状态缺少冻结证据或凭据")
	}
	return nil
}

func (c *QualificationCase) LatestObservations() []ObservationRevision {
	byKey := map[string]ObservationRevision{}
	for _, observation := range c.Observations {
		key := fmt.Sprintf("%03d/%03d", observation.ReplicateNo, observation.DayNo)
		if current, ok := byKey[key]; !ok || observation.Revision > current.Revision {
			byKey[key] = observation
		}
	}
	result := make([]ObservationRevision, 0, len(byKey))
	for _, observation := range byKey {
		result = append(result, observation)
	}
	SortObservations(result)
	return result
}

func (c *QualificationCase) OpenDeviations() []Deviation {
	result := make([]Deviation, 0)
	for _, item := range c.Deviations {
		if item.Status == "open" || item.Status == "in_progress" {
			result = append(result, item)
		}
	}
	return result
}
