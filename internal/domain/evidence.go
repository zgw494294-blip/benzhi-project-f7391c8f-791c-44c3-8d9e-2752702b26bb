package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type evidenceDigestInput struct {
	CaseID             string                `json:"caseId"`
	CaseVersion        int64                 `json:"caseVersion"`
	ProtocolCode       string                `json:"protocolCode"`
	Sampling           SamplingPlan          `json:"sampling"`
	Observations       []ObservationRevision `json:"observations"`
	Analysis           AnalysisSnapshot      `json:"analysis"`
	Deviations         []Deviation           `json:"deviations"`
	Reviews            []ReviewDecision      `json:"reviews"`
	SupplementalTrials []SupplementalTrial   `json:"supplementalTrials"`
}

func (c *QualificationCase) Freeze(bundleID, actor string, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusApproved {
		return invalidState(c.Status, "冻结证据")
	}
	if c.SamplingPlan == nil || c.Analysis == nil || len(c.Reviews) == 0 {
		return invalid("evidence", "抽样、分析或复核证据不完整")
	}
	if strings.TrimSpace(actor) == "" || bundleID == "" {
		return invalid("frozenBy", "冻结人和证据包 ID 不能为空")
	}
	observations := append([]ObservationRevision(nil), c.Observations...)
	SortObservations(observations)
	deviations := append([]Deviation(nil), c.Deviations...)
	sort.Slice(deviations, func(i, j int) bool { return deviations[i].ID < deviations[j].ID })
	reviews := append([]ReviewDecision(nil), c.Reviews...)
	sort.Slice(reviews, func(i, j int) bool { return reviews[i].DecidedAt.Before(reviews[j].DecidedAt) })
	trials := append([]SupplementalTrial(nil), c.SupplementalTrials...)
	sort.Slice(trials, func(i, j int) bool { return trials[i].ID < trials[j].ID })
	bundle := &EvidenceBundle{ID: bundleID, CaseID: c.ID, CaseVersion: c.Version, ProtocolCode: c.ProtocolCode, SamplingPlan: *c.SamplingPlan, Observations: observations, Analysis: *c.Analysis, Deviations: deviations, Reviews: reviews, SupplementalTrials: trials, EvidenceRefs: c.EvidenceReferences(), FrozenBy: strings.TrimSpace(actor), FrozenAt: now.UTC()}
	digest, err := RecalculateEvidenceDigest(bundle, c.ProtocolCode)
	if err != nil {
		return err
	}
	bundle.Digest = digest
	c.EvidenceBundle = bundle
	c.Status = StatusFrozen
	return nil
}

// RecalculateEvidenceDigest 是冻结与验真共同使用的唯一规范摘要算法。
func RecalculateEvidenceDigest(bundle *EvidenceBundle, protocolCode string) (string, error) {
	if bundle == nil {
		return "", invalid("evidenceBundle", "冻结证据包不存在")
	}
	observations := append([]ObservationRevision(nil), bundle.Observations...)
	SortObservations(observations)
	deviations := append([]Deviation(nil), bundle.Deviations...)
	sort.Slice(deviations, func(i, j int) bool { return deviations[i].ID < deviations[j].ID })
	reviews := append([]ReviewDecision(nil), bundle.Reviews...)
	sort.Slice(reviews, func(i, j int) bool {
		if reviews[i].DecidedAt.Equal(reviews[j].DecidedAt) {
			return reviews[i].ID < reviews[j].ID
		}
		return reviews[i].DecidedAt.Before(reviews[j].DecidedAt)
	})
	trials := append([]SupplementalTrial(nil), bundle.SupplementalTrials...)
	sort.Slice(trials, func(i, j int) bool { return trials[i].ID < trials[j].ID })
	for index := range trials {
		trials[index].Observations = append([]ObservationRevision(nil), trials[index].Observations...)
		SortObservations(trials[index].Observations)
		trials[index].AnalysisSnapshots = append([]AnalysisSnapshot(nil), trials[index].AnalysisSnapshots...)
		sort.Slice(trials[index].AnalysisSnapshots, func(i, j int) bool {
			return trials[index].AnalysisSnapshots[i].Sequence < trials[index].AnalysisSnapshots[j].Sequence
		})
	}
	sampling := bundle.SamplingPlan
	sampling.UnitQuotas = append([]SampleUnitQuota(nil), sampling.UnitQuotas...)
	for index := range sampling.UnitQuotas {
		sampling.UnitQuotas[index].ReplicateNos = append([]int(nil), sampling.UnitQuotas[index].ReplicateNos...)
		sort.Ints(sampling.UnitQuotas[index].ReplicateNos)
	}
	sort.Slice(sampling.UnitQuotas, func(i, j int) bool { return sampling.UnitQuotas[i].UnitName < sampling.UnitQuotas[j].UnitName })
	sampling.SampleUnits = append([]string(nil), sampling.SampleUnits...)
	sort.Strings(sampling.SampleUnits)
	input := evidenceDigestInput{CaseID: bundle.CaseID, CaseVersion: bundle.CaseVersion, ProtocolCode: protocolCode, Sampling: sampling, Observations: observations, Analysis: bundle.Analysis, Deviations: deviations, Reviews: reviews, SupplementalTrials: trials}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode evidence digest: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func (c *QualificationCase) EvidenceReferences() []string {
	refs := []string{}
	if c.SamplingPlan != nil {
		refs = append(refs, "sampling:"+c.SamplingPlan.ID)
	}
	for _, item := range c.Observations {
		refs = append(refs, fmt.Sprintf("observation:%s:r%d", item.ID, item.Revision))
	}
	for _, trial := range c.SupplementalTrials {
		refs = append(refs, "supplemental-trial:"+trial.ID)
		for _, item := range trial.Observations {
			refs = append(refs, fmt.Sprintf("observation:%s:r%d", item.ID, item.Revision))
		}
		for _, snapshot := range trial.AnalysisSnapshots {
			refs = append(refs, "analysis:"+snapshot.ID)
		}
	}
	if c.Analysis != nil {
		refs = append(refs, "analysis:"+c.Analysis.ID)
	}
	for _, item := range c.Deviations {
		refs = append(refs, "deviation:"+item.ID)
	}
	for _, item := range c.Reviews {
		refs = append(refs, "review:"+item.ID)
	}
	sort.Strings(refs)
	return refs
}

func (c *QualificationCase) IssueCredential(number, actor string, now time.Time) error {
	if c.Status == StatusCredentialIssued && c.Credential != nil {
		return nil
	}
	if c.Status != StatusFrozen || c.EvidenceBundle == nil {
		return invalidState(c.Status, "签发适格凭据")
	}
	if strings.TrimSpace(number) == "" || strings.TrimSpace(actor) == "" {
		return invalid("credential", "凭据编号和签发人不能为空")
	}
	c.Credential = &EligibilityCredential{CredentialNo: number, CaseID: c.ID, EvidenceBundleID: c.EvidenceBundle.ID, EvidenceDigest: c.EvidenceBundle.Digest, Decision: "eligible", ProtocolCode: c.ProtocolCode, IssuedBy: strings.TrimSpace(actor), IssuedAt: now.UTC()}
	c.Status = StatusCredentialIssued
	return nil
}

func CredentialNumber(caseID, digest string, issuedAt time.Time) string {
	short := digest
	if len(short) > 12 {
		short = short[:12]
	}
	id := strings.ReplaceAll(strings.ToUpper(caseID), "-", "")
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("SVG-%s-%s-%s", issuedAt.UTC().Format("20060102"), id, strings.ToUpper(short))
}
