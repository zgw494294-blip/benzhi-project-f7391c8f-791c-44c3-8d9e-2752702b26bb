package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

func (s *Store) Create(ctx context.Context, key, action, actor string, item *domain.QualificationCase) (*domain.QualificationCase, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if replay, exists, err := replayResult(ctx, tx, key, action); err != nil {
		return nil, false, err
	} else if exists {
		return replay, true, tx.Commit()
	}
	if err := item.Validate(); err != nil {
		return nil, false, err
	}
	data, err := encode(item)
	if err != nil {
		return nil, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cases(id, accession_code, status, version, aggregate_json, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, item.ID, item.AccessionCode, item.Status, item.Version, data, item.CreatedAt.Format(time.RFC3339Nano), item.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, false, fmt.Errorf("insert case: %w", err)
	}
	if err := appendAudit(ctx, tx, item, action, actor, nil); err != nil {
		return nil, false, err
	}
	if err := saveIdempotency(ctx, tx, key, action, item.ID, data); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	s.cacheCase(item)
	result, err := cloneCase(item)
	return result, false, err
}

func (s *Store) Update(ctx context.Context, caseID string, expectedVersion int64, key, action, actor string, auditDetails map[string]any, mutation application.Mutation) (*domain.QualificationCase, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if replay, exists, err := replayResult(ctx, tx, key, action); err != nil {
		return nil, false, err
	} else if exists {
		if replay.ID != caseID {
			return nil, false, application.ErrIdempotencyConflict
		}
		return replay, true, tx.Commit()
	}
	item, ok := s.cachedCase(caseID)
	if !ok {
		item, err = loadCaseTx(ctx, tx, caseID)
		if err != nil {
			return nil, false, err
		}
		s.cacheCase(item)
	}
	if item.Version != expectedVersion {
		return nil, false, application.ErrVersionConflict
	}
	if err := mutation(item); err != nil {
		return nil, false, err
	}
	item.Version++
	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return nil, false, err
	}
	data, err := encode(item)
	if err != nil {
		return nil, false, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE cases SET status=?, version=?, aggregate_json=?, updated_at=? WHERE id=? AND version=?`, item.Status, item.Version, data, item.UpdatedAt.Format(time.RFC3339Nano), item.ID, expectedVersion)
	if err != nil {
		return nil, false, fmt.Errorf("conditional case update: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if changed != 1 {
		return nil, false, application.ErrVersionConflict
	}
	if err := syncEvidence(ctx, tx, item); err != nil {
		return nil, false, err
	}
	if err := appendAudit(ctx, tx, item, action, actor, auditDetails); err != nil {
		return nil, false, err
	}
	if err := saveIdempotency(ctx, tx, key, action, item.ID, data); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	s.cacheCase(item)
	resultCase, err := cloneCase(item)
	return resultCase, false, err
}

func replayResult(ctx context.Context, tx *sql.Tx, key, action string) (*domain.QualificationCase, bool, error) {
	var storedAction string
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT action, response_json FROM idempotency_results WHERE idempotency_key=?`, key).Scan(&storedAction, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if storedAction != action {
		return nil, false, application.ErrIdempotencyConflict
	}
	item, err := decodeCase(data)
	return item, true, err
}

func saveIdempotency(ctx context.Context, tx *sql.Tx, key, action, caseID string, data []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO idempotency_results(idempotency_key, action, case_id, response_json, created_at) VALUES(?, ?, ?, ?, ?)`, key, action, caseID, data, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save idempotency result: %w", err)
	}
	return nil
}

func appendAudit(ctx context.Context, tx *sql.Tx, item *domain.QualificationCase, action, actor string, extra map[string]any) error {
	values := map[string]any{"version": item.Version, "status": item.Status}
	for key, value := range extra {
		values[key] = value
	}
	details, _ := encode(values)
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(case_id, action, actor, details_json, occurred_at) VALUES(?, ?, ?, ?, ?)`, item.ID, action, actor, details, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

func loadCaseTx(ctx context.Context, tx *sql.Tx, caseID string) (*domain.QualificationCase, error) {
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT aggregate_json FROM cases WHERE id=?`, caseID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeCase(data)
}
