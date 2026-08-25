package analysis

import (
	"seed-vigor-gate/internal/domain"
	"testing"
	"time"
)

func TestEvaluateEligible(t *testing.T) {
	c := &domain.QualificationCase{ID: "case-1", ProtocolCode: "ISTA-2025", SamplingPlan: &domain.SamplingPlan{ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25}}
	for replicate := 1; replicate <= 2; replicate++ {
		c.Observations = append(c.Observations, domain.ObservationRevision{ID: "obs", ReplicateNo: replicate, DayNo: 7, Revision: 1, NormalCount: 90 + replicate, AbnormalCount: 4, UngerminatedCount: 6 - replicate, TemperatureC: 22, Submitted: true, RecordedAt: time.Now()})
	}
	result := Evaluate(c)
	if len(result.Findings) != 0 || result.Metric.CandidateDecision != "eligible" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Metric.EffectiveSampleSize != 200 {
		t.Fatalf("effective size = %d", result.Metric.EffectiveSampleSize)
	}
}

func TestEvaluateFindsMissingAndEnvironment(t *testing.T) {
	c := &domain.QualificationCase{ID: "case-1", ProtocolCode: "ISTA-2025", SamplingPlan: &domain.SamplingPlan{ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25}, Observations: []domain.ObservationRevision{{ID: "o1", ReplicateNo: 1, DayNo: 7, Revision: 1, NormalCount: 80, AbnormalCount: 10, UngerminatedCount: 5, ContaminatedCount: 5, TemperatureC: 30, Submitted: true}}}
	result := Evaluate(c)
	if len(result.Findings) < 2 {
		t.Fatalf("expected multiple findings: %+v", result.Findings)
	}
}

func TestEvaluateTemporalTrajectoryUsesLatestFormalRevision(t *testing.T) {
	c := &domain.QualificationCase{ID: "case-time", ProtocolCode: "ISTA-2025", SamplingPlan: &domain.SamplingPlan{ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25}}
	c.Observations = []domain.ObservationRevision{
		{ID: "r1d3", ReplicateNo: 1, DayNo: 3, Revision: 1, NormalCount: 35, AbnormalCount: 5, UngerminatedCount: 60, TemperatureC: 22, Submitted: true},
		{ID: "r1d5", ReplicateNo: 1, DayNo: 5, Revision: 1, NormalCount: 30, AbnormalCount: 5, UngerminatedCount: 65, TemperatureC: 22, Submitted: true},
		{ID: "draft", ReplicateNo: 1, DayNo: 5, Revision: 2, NormalCount: 90, AbnormalCount: 5, UngerminatedCount: 5, TemperatureC: 22, Submitted: false},
		{ID: "r2d5", ReplicateNo: 2, DayNo: 5, Revision: 1, NormalCount: 90, AbnormalCount: 5, UngerminatedCount: 5, TemperatureC: 22, Submitted: true},
	}
	result := Evaluate(c)
	codes := map[string]bool{}
	for _, finding := range result.Findings {
		codes[finding.RuleCode] = true
	}
	if !codes["GERMINATION_CUMULATIVE_DECREASE"] || !codes["UNGERMINATED_COUNT_INCREASE"] || len(result.Trajectories) != 3 {
		t.Fatalf("unexpected temporal result: %+v", result)
	}
	if result.Trajectories[1].ObservationRef != "observation:r1d5:r1" {
		t.Fatalf("draft revision affected trajectory: %+v", result.Trajectories)
	}
}
