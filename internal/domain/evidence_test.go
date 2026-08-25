package domain

import (
	"testing"
	"time"
)

func TestFreezeDigestIsStable(t *testing.T) {
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	build := func() *QualificationCase {
		item := validCase(t)
		item.Status = StatusApproved
		item.SamplingPlan = &SamplingPlan{ID: "sample", CaseID: item.ID, Revision: 1, SampleUnits: []string{"A", "B"}, ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25}
		item.Observations = []ObservationRevision{{ID: "o2", ReplicateNo: 2, DayNo: 7, Revision: 1}, {ID: "o1", ReplicateNo: 1, DayNo: 7, Revision: 1}}
		item.Analysis = &AnalysisSnapshot{ID: "analysis", Metric: Metric{GerminationRate: 90}}
		item.Reviews = []ReviewDecision{{ID: "review", Decision: "approve", DecidedAt: now}}
		return item
	}
	first, second := build(), build()
	if err := first.Freeze("bundle-a", "复核员", now); err != nil {
		t.Fatal(err)
	}
	if err := second.Freeze("bundle-b", "复核员", now); err != nil {
		t.Fatal(err)
	}
	if first.EvidenceBundle.Digest != second.EvidenceBundle.Digest {
		t.Fatalf("digest not stable: %s != %s", first.EvidenceBundle.Digest, second.EvidenceBundle.Digest)
	}
	if first.EvidenceBundle.Observations[0].ReplicateNo != 1 {
		t.Fatal("evidence was not sorted")
	}
	recalculated, err := RecalculateEvidenceDigest(first.EvidenceBundle, first.ProtocolCode)
	if err != nil || recalculated != first.EvidenceBundle.Digest {
		t.Fatalf("recalculated digest mismatch: %s %v", recalculated, err)
	}
	first.EvidenceBundle.Observations[0].NormalCount++
	tampered, err := RecalculateEvidenceDigest(first.EvidenceBundle, first.ProtocolCode)
	if err != nil || tampered == first.EvidenceBundle.Digest {
		t.Fatalf("tampered evidence was not detected: %s %v", tampered, err)
	}
}
