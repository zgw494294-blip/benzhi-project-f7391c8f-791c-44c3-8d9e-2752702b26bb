package wrapped_read_error_chain_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
	"seed-vigor-gate/internal/httpapi"
)

type failureStage string

const (
	stageWorkbenchCase      failureStage = "workbench-case"
	stageWorkbenchTimeline  failureStage = "workbench-timeline"
	stageList               failureStage = "list"
	stageTimeline           failureStage = "timeline"
	stageCredential         failureStage = "credential"
	stageEvidenceBundle     failureStage = "evidence-bundle"
	stageCredentialTimeline failureStage = "credential-timeline"
)

type wrappedNotFoundRepository struct {
	stage failureStage
}

func wrappedNotFound(operation string) error {
	return fmt.Errorf("%s from repository: %w", operation, application.ErrNotFound)
}

func (r *wrappedNotFoundRepository) Create(context.Context, string, string, string, *domain.QualificationCase) (*domain.QualificationCase, bool, error) {
	panic("unexpected Create call")
}

func (r *wrappedNotFoundRepository) Update(context.Context, string, int64, string, string, string, map[string]any, application.Mutation) (*domain.QualificationCase, bool, error) {
	panic("unexpected Update call")
}

func (r *wrappedNotFoundRepository) Get(context.Context, string) (*domain.QualificationCase, error) {
	if r.stage == stageWorkbenchCase {
		return nil, wrappedNotFound("get case")
	}
	return &domain.QualificationCase{ID: "case-1", Status: domain.StatusDraft}, nil
}

func (r *wrappedNotFoundRepository) List(context.Context, string, int) ([]domain.QualificationCase, error) {
	if r.stage == stageList {
		return nil, wrappedNotFound("list cases")
	}
	return []domain.QualificationCase{}, nil
}

func (r *wrappedNotFoundRepository) Timeline(context.Context, string) ([]domain.AuditEvent, error) {
	switch r.stage {
	case stageWorkbenchTimeline, stageTimeline, stageCredentialTimeline:
		return nil, wrappedNotFound("load timeline")
	default:
		return []domain.AuditEvent{}, nil
	}
}

func (r *wrappedNotFoundRepository) GetCredential(context.Context, string) (*domain.EligibilityCredential, error) {
	if r.stage == stageCredential {
		return nil, wrappedNotFound("load credential")
	}
	return &domain.EligibilityCredential{
		CredentialNo:     "CRED-1",
		CaseID:           "case-1",
		EvidenceBundleID: "bundle-1",
	}, nil
}

func (r *wrappedNotFoundRepository) GetEvidenceBundle(context.Context, string) (*domain.EvidenceBundle, error) {
	if r.stage == stageEvidenceBundle {
		return nil, wrappedNotFound("load evidence bundle")
	}
	return &domain.EvidenceBundle{ID: "bundle-1", CaseID: "case-1"}, nil
}

func (r *wrappedNotFoundRepository) IntegrityCheck(context.Context) error { return nil }
func (r *wrappedNotFoundRepository) Close() error                         { return nil }

func TestWrappedRepositoryErrorsRemainClassifiable(t *testing.T) {
	tests := []struct {
		name  string
		stage failureStage
		path  string
	}{
		{name: "workbench case", stage: stageWorkbenchCase, path: "/api/cases/case-1"},
		{name: "workbench timeline", stage: stageWorkbenchTimeline, path: "/api/cases/case-1"},
		{name: "case list", stage: stageList, path: "/api/cases?limit=10"},
		{name: "direct timeline", stage: stageTimeline, path: "/api/cases/case-1/timeline"},
		{name: "credential", stage: stageCredential, path: "/api/credentials/CRED-1/verify"},
		{name: "evidence bundle", stage: stageEvidenceBundle, path: "/api/credentials/CRED-1/verify"},
		{name: "credential timeline", stage: stageCredentialTimeline, path: "/api/credentials/CRED-1/verify"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &wrappedNotFoundRepository{stage: test.stage}
			server := httpapi.New(application.NewService(repository))
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			server.Handler().ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("wrapped ErrNotFound must remain HTTP 404, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}
