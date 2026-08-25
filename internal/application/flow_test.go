package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
	"seed-vigor-gate/internal/store"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type sequenceIDs struct{ next int }

func (ids *sequenceIDs) New(prefix string) string {
	ids.next++
	return fmt.Sprintf("%s%03d", prefix, ids.next)
}

func TestBatchAndSupplementalTrialClosure(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service := application.NewServiceWithDependencies(repository, fixedClock{now: time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)}, &sequenceIDs{})
	ctx := context.Background()
	item, err := service.CreateCase(ctx, application.CreateCaseCommand{IdempotencyKey: "flow-create-001", Actor: "接收员", AccessionCode: "FLOW-001", Source: "闭环测试资源圃", HarvestedAt: "2026-08-20", DeclaredSeedCount: 500, ProtocolCode: "ISTA-2025"})
	if err != nil {
		t.Fatal(err)
	}
	item, err = service.ConfirmSampling(ctx, item.ID, application.ConfirmSamplingCommand{WriteMeta: application.WriteMeta{ExpectedVersion: item.Version, IdempotencyKey: "flow-sampling-01", Actor: "接收员"}, UnitQuotas: []domain.SampleUnitQuota{{UnitName: "A", PlannedCount: 100, ReplicateNos: []int{1}}, {UnitName: "B", PlannedCount: 100, ReplicateNos: []int{2}}}, ReplicateCount: 2, SeedsPerReplicate: 100, TemperatureMinC: 20, TemperatureMaxC: 25})
	if err != nil {
		t.Fatal(err)
	}
	batchCommand := application.RecordObservationCommand{WriteMeta: application.WriteMeta{ExpectedVersion: item.Version, IdempotencyKey: "flow-observe-001", Actor: "试验员"}, Rows: []application.ObservationRowCommand{{ReplicateNo: 1, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 30, Submitted: true}, {ReplicateNo: 2, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 22, Submitted: true}}}
	batch, err := service.RecordObservation(ctx, item.ID, batchCommand)
	if err != nil || batch.AcceptedRows != 2 || batch.Version != item.Version+1 {
		t.Fatalf("batch result: %+v err=%v", batch, err)
	}
	replay, err := service.RecordObservation(ctx, item.ID, batchCommand)
	if err != nil || fmt.Sprint(replay.Revisions) != fmt.Sprint(batch.Revisions) {
		t.Fatalf("idempotent replay changed revisions: %+v %+v err=%v", batch, replay, err)
	}
	item, err = service.Analyze(ctx, item.ID, application.AnalyzeCommand{WriteMeta: application.WriteMeta{ExpectedVersion: batch.Version, IdempotencyKey: "flow-analysis-01", Actor: "试验员"}})
	if err != nil || len(item.Deviations) != 1 {
		t.Fatalf("analysis: %+v err=%v", item, err)
	}
	item, err = service.ResolveDeviation(ctx, item.ID, item.Deviations[0].ID, application.ResolveDeviationCommand{WriteMeta: application.WriteMeta{ExpectedVersion: item.Version, IdempotencyKey: "flow-resolve-001", Actor: "试验员"}, Cause: "培养箱设定偏移", CorrectiveAction: "校准后重做", StartSupplementalTrial: true})
	if err != nil || len(item.SupplementalTrials) != 1 {
		t.Fatalf("init supplemental trial: %+v err=%v", item, err)
	}
	trialID := item.SupplementalTrials[0].ID
	trialBatch, err := service.RecordObservation(ctx, item.ID, application.RecordObservationCommand{WriteMeta: application.WriteMeta{ExpectedVersion: item.Version, IdempotencyKey: "flow-trial-obs1", Actor: "试验员"}, SupplementalTrialID: trialID, Rows: []application.ObservationRowCommand{{ReplicateNo: 1, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 22, Submitted: true}, {ReplicateNo: 2, DayNo: 7, NormalCount: 90, AbnormalCount: 4, UngerminatedCount: 6, TemperatureC: 22, Submitted: true}}})
	if err != nil {
		t.Fatal(err)
	}
	item, err = service.Analyze(ctx, item.ID, application.AnalyzeCommand{WriteMeta: application.WriteMeta{ExpectedVersion: trialBatch.Version, IdempotencyKey: "flow-trial-analysis", Actor: "试验员"}, SupplementalTrialID: trialID})
	if err != nil || item.Deviations[0].Status != "resolved" || item.SupplementalTrials[0].Status != "completed" {
		t.Fatalf("supplemental closure: %+v err=%v", item, err)
	}
}
