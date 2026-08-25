package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"seed-vigor-gate/internal/domain"
)

func (s *Service) GetCase(ctx context.Context, caseID string) (*domain.QualificationCase, error) {
	item, err := s.repository.Get(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get case: %v", err)
	}
	return item, nil
}
func (s *Service) ListCases(ctx context.Context, status string, limit int) ([]domain.QualificationCase, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	items, err := s.repository.List(ctx, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list cases: %v", err)
	}
	return items, nil
}
func (s *Service) Timeline(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	events, err := s.repository.Timeline(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("load timeline: %v", err)
	}
	return events, nil
}

func (s *Service) Workbench(ctx context.Context, caseID string) (*WorkbenchCase, error) {
	item, err := s.repository.Get(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("load workbench case: %v", err)
	}
	events, err := s.repository.Timeline(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("load workbench timeline: %v", err)
	}
	return &WorkbenchCase{Case: item, Timeline: events, NextActions: nextActions(item)}, nil
}

func nextActions(c *domain.QualificationCase) []string {
	switch c.Status {
	case domain.StatusDraft:
		return []string{"confirm_sampling"}
	case domain.StatusSamplingConfirmed, domain.StatusObserving, domain.StatusReturned:
		return []string{"record_observation", "analyze", "resolve_deviation"}
	case domain.StatusPendingReview:
		return []string{"review_approve", "review_return"}
	case domain.StatusApproved:
		return []string{"freeze"}
	case domain.StatusFrozen:
		return []string{"issue_credential"}
	case domain.StatusCredentialIssued:
		return []string{"verify_credential"}
	default:
		return []string{}
	}
}

func (s *Service) VerifyCredential(ctx context.Context, number string) (*CredentialVerification, error) {
	credential, err := s.repository.GetCredential(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("load credential: %v", err)
	}
	bundle, err := s.repository.GetEvidenceBundle(ctx, credential.EvidenceBundleID)
	if err != nil {
		return nil, fmt.Errorf("load evidence bundle: %v", err)
	}
	timeline, err := s.repository.Timeline(ctx, credential.CaseID)
	if err != nil {
		return nil, fmt.Errorf("load credential timeline: %v", err)
	}
	recalculated, digestErr := domain.RecalculateEvidenceDigest(bundle, bundle.ProtocolCode)
	checks := make([]VerificationCheck, 0, 9)
	add := func(code string, passed bool, message string, refs ...string) {
		checks = append(checks, VerificationCheck{Code: code, Passed: passed, Message: message, EvidenceRefs: refs})
	}
	add("BUNDLE_DIGEST_RECALCULATED", digestErr == nil && recalculated == bundle.Digest, "重算摘要与冻结包存储摘要一致", "bundle:"+bundle.ID)
	add("CREDENTIAL_DIGEST_MATCH", credential.EvidenceDigest == bundle.Digest, "凭据摘要与冻结包摘要一致", "credential:"+credential.CredentialNo, "bundle:"+bundle.ID)
	add("CASE_RELATION_MATCH", credential.CaseID == bundle.CaseID, "凭据批次与冻结包批次一致", "case:"+bundle.CaseID)
	add("BUNDLE_RELATION_MATCH", credential.EvidenceBundleID == bundle.ID, "凭据关联正确的冻结包", "bundle:"+bundle.ID)
	add("PROTOCOL_MATCH", bundle.ProtocolCode != "" && credential.ProtocolCode == bundle.ProtocolCode, "凭据协议与冻结证据采用的协议一致", "credential:"+credential.CredentialNo, "bundle:"+bundle.ID)
	add("DECISION_MATCH", credential.Decision == "eligible" && bundle.Analysis.Metric.CandidateDecision == "eligible" && bundleApproved(bundle), "凭据适格决定与采用分析及复核批准一致", "analysis:"+bundle.Analysis.ID)
	approve, freeze, issue := auditMilestones(timeline)
	ordered := approve != nil && freeze != nil && issue != nil && approve.Sequence < freeze.Sequence && freeze.Sequence < issue.Sequence
	add("AUDIT_EVENT_ORDER", ordered, "批准、冻结和签发事件顺序正确", auditRefs(approve, freeze, issue)...)
	versions := ordered && auditVersion(approve) == bundle.CaseVersion && auditVersion(freeze) == bundle.CaseVersion+1 && auditVersion(issue) == bundle.CaseVersion+2
	add("AUDIT_VERSION_LINK", versions, "审计事件版本与冻结版本连续关联", auditRefs(approve, freeze, issue)...)
	valid := true
	for _, check := range checks {
		if !check.Passed {
			valid = false
		}
	}
	message := "凭据有效，冻结内容重算与全部关系检查通过"
	if !valid {
		message = "凭据无效，冻结摘要或关系检查未通过"
	}
	return &CredentialVerification{Valid: valid, Credential: credential, Bundle: bundle, Timeline: timeline, Message: message, StoredDigest: bundle.Digest, RecalculatedDigest: recalculated, Checks: checks}, nil
}

func bundleApproved(bundle *domain.EvidenceBundle) bool {
	for _, review := range bundle.Reviews {
		if review.Decision == "approve" && review.AnalysisID == bundle.Analysis.ID {
			return true
		}
	}
	return false
}

func auditMilestones(events []domain.AuditEvent) (approve, freeze, issue *domain.AuditEvent) {
	for index := range events {
		event := &events[index]
		switch event.Action {
		case "review.approve":
			approve = event
		case "evidence.frozen":
			freeze = event
		case "credential.issued":
			issue = event
		}
	}
	return
}

func auditVersion(event *domain.AuditEvent) int64 {
	if event == nil {
		return 0
	}
	switch value := event.Details["version"].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func auditRefs(events ...*domain.AuditEvent) []string {
	refs := make([]string, 0, len(events))
	for _, event := range events {
		if event != nil {
			refs = append(refs, fmt.Sprintf("audit:%d", event.Sequence))
		}
	}
	return refs
}

func StableJSONDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
