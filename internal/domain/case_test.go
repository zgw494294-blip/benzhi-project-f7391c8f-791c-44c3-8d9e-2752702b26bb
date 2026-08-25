package domain

import (
	"errors"
	"testing"
	"time"
)

func validCase(t *testing.T) *QualificationCase {
	t.Helper()
	item, err := CreateCase(NewCase{ID: "case-test", AccessionCode: "ACC-TEST-01", Source: "国家种质资源圃", HarvestedAt: "2025-09-01", DeclaredSeedCount: 500, ProtocolCode: "ISTA-2025", Now: time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestSamplingAndObservationRevision(t *testing.T) {
	item := validCase(t)
	now := time.Now().UTC()
	err := item.ConfirmSampling(ConfirmSamplingInput{ID: "sampling-1", SampleUnits: []string{"A", "B", "C", "D"}, ReplicateCount: 4, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25, ConfirmedBy: "接收员", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	first, err := item.RecordObservation(RecordObservationInput{ID: "obs-1", ReplicateNo: 1, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 22, RecordedBy: "试验员", Submitted: false, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	second, err := item.RecordObservation(RecordObservationInput{ID: "obs-2", ReplicateNo: 1, DayNo: 7, NormalCount: 91, AbnormalCount: 4, UngerminatedCount: 5, TemperatureC: 22, RecordedBy: "试验员", Submitted: true, Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 || second.Revision != 2 || len(item.Observations) != 2 {
		t.Fatalf("revision history lost: %+v", item.Observations)
	}
	if item.Observations[0].NormalCount != 90 {
		t.Fatal("earlier revision was overwritten")
	}
}

func TestCountConservation(t *testing.T) {
	item := validCase(t)
	now := time.Now().UTC()
	_ = item.ConfirmSampling(ConfirmSamplingInput{ID: "sampling-1", SampleUnits: []string{"A", "B"}, ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25, ConfirmedBy: "接收员", Now: now})
	_, err := item.RecordObservation(RecordObservationInput{ID: "obs-1", ReplicateNo: 1, DayNo: 7, NormalCount: 90, AbnormalCount: 2, UngerminatedCount: 2, TemperatureC: 22, RecordedBy: "试验员", Submitted: true, Now: now})
	var issue *DomainError
	if !errors.As(err, &issue) || issue.Field != "counts" {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestSamplingQuotaAndAtomicObservationBatch(t *testing.T) {
	now := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	item, err := CreateCase(NewCase{ID: "case-quota", AccessionCode: "ACC-QUOTA", Source: "配额测试资源圃", HarvestedAt: "2026-08-20", DeclaredSeedCount: 500, ProtocolCode: "ISTA-2025", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	err = item.ConfirmSampling(ConfirmSamplingInput{ID: "sampling-quota", ReplicateCount: 4, SeedsPerReplicate: 100, UnitQuotas: []SampleUnitQuota{{UnitName: "D", PlannedCount: 100, ReplicateNos: []int{4}}, {UnitName: "A", PlannedCount: 100, ReplicateNos: []int{1}}, {UnitName: "C", PlannedCount: 100, ReplicateNos: []int{3}}, {UnitName: "B", PlannedCount: 100, ReplicateNos: []int{2}}}, TemperatureMinC: 20, TemperatureMaxC: 25, ConfirmedBy: "接收员", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if item.SamplingPlan.TotalSampled != 400 || item.SamplingPlan.RemainingReserve != 100 || item.SamplingPlan.UnitQuotas[0].UnitName != "A" {
		t.Fatalf("unexpected plan: %+v", item.SamplingPlan)
	}
	before := len(item.Observations)
	_, err = item.RecordObservations([]RecordObservationInput{{ID: "o1", ReplicateNo: 1, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 22, Submitted: true, RecordedBy: "试验员", Now: now}, {ID: "o2", ReplicateNo: 2, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 5, TemperatureC: 22, Submitted: true, RecordedBy: "试验员", Now: now}})
	if err == nil || len(item.Observations) != before {
		t.Fatalf("invalid batch must be atomic: err=%v observations=%d", err, len(item.Observations))
	}
}

func TestFrozenCaseRejectsMutation(t *testing.T) {
	item := validCase(t)
	now := time.Now().UTC()
	item.Status = StatusFrozen
	item.EvidenceBundle = &EvidenceBundle{ID: "bundle", Digest: "digest"}
	err := item.ConfirmSampling(ConfirmSamplingInput{ID: "sampling", Now: now})
	var issue *DomainError
	if !errors.As(err, &issue) || issue.Code != CodeFrozen {
		t.Fatalf("expected frozen error, got %v", err)
	}
}
