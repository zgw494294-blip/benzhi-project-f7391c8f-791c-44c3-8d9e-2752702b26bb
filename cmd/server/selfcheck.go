package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

func RunSelfCheck(config Config) error {
	temporary, err := os.MkdirTemp("", "seed-vigor-gate-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	config.DatabasePath = filepath.Join(temporary, "selfcheck.db")
	runtime, err := BuildRuntime(config)
	if err != nil {
		return err
	}
	errors := make(chan error, 1)
	go runtime.Serve(errors)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	defer func() {
		shutdownCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		_ = runtime.Shutdown(shutdownCtx)
	}()
	client := &http.Client{Timeout: 3 * time.Second}
	base := "http://" + config.Address
	if err := waitReady(ctx, client, base); err != nil {
		return err
	}
	created, err := selfPost[domain.QualificationCase](ctx, client, base+"/api/cases", map[string]any{"idempotencyKey": "self-create-0001", "actor": "自检接收员", "accessionCode": "SELF-ACC-001", "source": "自检资源圃代表性混合样", "harvestedAt": "2025-10-12", "declaredSeedCount": 500, "protocolCode": "ISTA-2025"})
	if err != nil {
		return err
	}
	item, err := selfPost[domain.QualificationCase](ctx, client, base+"/api/cases/"+created.ID+"/sampling-plan/confirm", map[string]any{"expectedVersion": created.Version, "idempotencyKey": "self-sample-0001", "actor": "自检接收员", "unitQuotas": []map[string]any{{"unitName": "袋位-A", "plannedCount": 100, "replicateNos": []int{1}}, {"unitName": "袋位-B", "plannedCount": 100, "replicateNos": []int{2}}, {"unitName": "袋位-C", "plannedCount": 100, "replicateNos": []int{3}}, {"unitName": "袋位-D", "plannedCount": 100, "replicateNos": []int{4}}}, "replicateCount": 4, "seedsPerReplicate": 100, "temperatureMinC": 20, "temperatureMaxC": 25, "environmentNotes": "恒温培养箱纸床法"})
	if err != nil {
		return err
	}
	rows := make([]map[string]any, 0, 4)
	for replicate := 1; replicate <= 4; replicate++ {
		normal := 89 + replicate
		abnormal := 4
		ungerminated := 100 - normal - abnormal
		rows = append(rows, map[string]any{"replicateNo": replicate, "dayNo": 7, "normalCount": normal, "abnormalCount": abnormal, "ungerminatedCount": ungerminated, "contaminatedCount": 0, "temperatureC": 22.0, "submitted": true})
	}
	batch, err := selfPost[application.ObservationBatchResult](ctx, client, base+"/api/cases/"+created.ID+"/observations", map[string]any{"expectedVersion": item.Version, "idempotencyKey": "self-observe-batch-0001", "actor": "自检试验员", "rows": rows})
	if err != nil {
		return err
	}
	if batch.AcceptedRows != 4 || batch.SubmittedCount != 4 || batch.Version != item.Version+1 {
		return fmt.Errorf("observation batch result invalid: %+v", batch)
	}
	workbench, err := selfGet[application.WorkbenchCase](ctx, client, base+"/api/cases/"+created.ID)
	if err != nil {
		return err
	}
	item = *workbench.Case
	item, err = selfPost[domain.QualificationCase](ctx, client, base+"/api/cases/"+created.ID+"/analysis", metaBody(item.Version, "self-analysis-01", "自检试验员"))
	if err != nil {
		return err
	}
	if item.Status != domain.StatusPendingReview || item.Analysis == nil || len(item.Analysis.Findings) != 0 {
		return fmt.Errorf("analysis did not reach review: status=%s findings=%v", item.Status, item.Analysis)
	}
	item, err = selfPost[domain.QualificationCase](ctx, client, base+"/api/cases/"+created.ID+"/review", merge(metaBody(item.Version, "self-review-0001", "自检复核员"), map[string]any{"decision": "approve", "reason": "计数、环境与组间差均满足协议"}))
	if err != nil {
		return err
	}
	item, err = selfPost[domain.QualificationCase](ctx, client, base+"/api/cases/"+created.ID+"/freeze", metaBody(item.Version, "self-freeze-0001", "自检复核员"))
	if err != nil {
		return err
	}
	if item.EvidenceBundle == nil || len(item.EvidenceBundle.Digest) != 64 {
		return fmt.Errorf("invalid frozen evidence digest")
	}
	credential, err := selfPost[domain.EligibilityCredential](ctx, client, base+"/api/cases/"+created.ID+"/credential", metaBody(item.Version, "self-issue-00001", "自检复核员"))
	if err != nil {
		return err
	}
	verification, err := selfGet[application.CredentialVerification](ctx, client, base+"/api/credentials/"+credential.CredentialNo+"/verify")
	if err != nil {
		return err
	}
	if !verification.Valid || verification.Credential.EvidenceDigest != item.EvidenceBundle.Digest || len(verification.Timeline) < 7 || verification.StoredDigest != verification.RecalculatedDigest {
		return fmt.Errorf("credential verification failed: %+v", verification)
	}
	for _, check := range verification.Checks {
		if !check.Passed {
			return fmt.Errorf("credential verification check failed: %+v", check)
		}
	}
	select {
	case serveErr := <-errors:
		if serveErr != nil {
			return serveErr
		}
	default:
	}
	return nil
}

func waitReady(ctx context.Context, client *http.Client, base string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/readyz", nil)
		response, err := client.Do(request)
		if err == nil {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func selfPost[T any](ctx context.Context, client *http.Client, url string, body any) (T, error) {
	var zero T
	data, err := json.Marshal(body)
	if err != nil {
		return zero, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return zero, err
	}
	request.Header.Set("Content-Type", "application/json")
	return selfDo[T](client, request)
}
func selfGet[T any](ctx context.Context, client *http.Client, url string) (T, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		var zero T
		return zero, err
	}
	return selfDo[T](client, request)
}
func selfDo[T any](client *http.Client, request *http.Request) (T, error) {
	var zero T
	response, err := client.Do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return zero, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return zero, fmt.Errorf("%s %s returned %d: %s", request.Method, request.URL.Path, response.StatusCode, string(data))
	}
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return zero, err
	}
	return envelope.Data, nil
}
func metaBody(version int64, key, actor string) map[string]any {
	return map[string]any{"expectedVersion": version, "idempotencyKey": key, "actor": actor}
}
func merge(left, right map[string]any) map[string]any {
	for key, value := range right {
		left[key] = value
	}
	return left
}
