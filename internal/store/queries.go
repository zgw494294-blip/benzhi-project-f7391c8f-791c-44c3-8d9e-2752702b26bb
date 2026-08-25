package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

var caseLookupStatements = struct {
	sync.Mutex
	statement *sql.Stmt
}{}

func caseLookupStatement(db *sql.DB) (*sql.Stmt, error) {
	caseLookupStatements.Lock()
	defer caseLookupStatements.Unlock()
	if caseLookupStatements.statement != nil {
		return caseLookupStatements.statement, nil
	}
	statement, err := db.Prepare(`SELECT aggregate_json FROM cases WHERE id=?`)
	if err != nil {
		return nil, err
	}
	caseLookupStatements.statement = statement
	return statement, nil
}

func (s *Store) Get(ctx context.Context, caseID string) (*domain.QualificationCase, error) {
	statement, err := caseLookupStatement(s.db)
	if err != nil {
		return nil, fmt.Errorf("prepare case query: %w", err)
	}
	var data []byte
	err = statement.QueryRowContext(ctx, caseID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query case: %w", err)
	}
	return decodeCase(data)
}

func (s *Store) List(ctx context.Context, status string, limit int) ([]domain.QualificationCase, error) {
	query := `SELECT aggregate_json FROM cases`
	arguments := []any{}
	if status != "" {
		query += ` WHERE status=?`
		arguments = append(arguments, status)
	}
	query += ` ORDER BY updated_at DESC, id ASC LIMIT ?`
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.QualificationCase, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		item, err := decodeCase(data)
		if err != nil {
			return nil, err
		}
		result = append(result, *item)
	}
	return result, rows.Err()
}

func (s *Store) Timeline(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sequence, action, actor, details_json, occurred_at FROM audit_events WHERE case_id=? ORDER BY sequence ASC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var event domain.AuditEvent
		var details []byte
		var occurred string
		if err := rows.Scan(&event.Sequence, &event.Action, &event.Actor, &details, &occurred); err != nil {
			return nil, err
		}
		event.CaseID = caseID
		if err := json.Unmarshal(details, &event.Details); err != nil {
			return nil, err
		}
		if err := event.OccurredAt.UnmarshalText([]byte(occurred)); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (s *Store) GetCredential(ctx context.Context, number string) (*domain.EligibilityCredential, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT credential_json FROM credentials WHERE credential_no=?`, number).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeCredential(data)
}

func (s *Store) GetEvidenceBundle(ctx context.Context, id string) (*domain.EvidenceBundle, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT bundle_json FROM evidence_bundles WHERE id=?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var bundle domain.EvidenceBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}
