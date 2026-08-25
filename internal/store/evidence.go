package store

import (
	"context"
	"database/sql"
	"fmt"

	"seed-vigor-gate/internal/domain"
)

func syncEvidence(ctx context.Context, tx *sql.Tx, item *domain.QualificationCase) error {
	for _, observation := range item.Observations {
		data, err := encode(observation)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO observation_revisions(id, case_id, replicate_no, day_no, revision, evidence_json, recorded_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, observation.ID, item.ID, observation.ReplicateNo, observation.DayNo, observation.Revision, data, observation.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
		if err != nil {
			return fmt.Errorf("append observation revision: %w", err)
		}
	}
	for _, deviation := range item.Deviations {
		data, err := encode(deviation)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO deviations(id, case_id, rule_code, status, evidence_json) VALUES(?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET status=excluded.status, evidence_json=excluded.evidence_json`, deviation.ID, item.ID, deviation.RuleCode, deviation.Status, data)
		if err != nil {
			return fmt.Errorf("save deviation: %w", err)
		}
	}
	for trialIndex := range item.SupplementalTrials {
		trial := &item.SupplementalTrials[trialIndex]
		data, err := encode(trial)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO supplemental_trials(id, case_id, deviation_id, status, trial_json, initiated_at) VALUES(?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET status=excluded.status, trial_json=excluded.trial_json`, trial.ID, item.ID, trial.DeviationID, trial.Status, data, trial.InitiatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
		if err != nil {
			return fmt.Errorf("save supplemental trial: %w", err)
		}
		for _, observation := range trial.Observations {
			observationData, err := encode(observation)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO supplemental_observation_revisions(id, case_id, supplemental_trial_id, replicate_no, day_no, revision, evidence_json, recorded_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, observation.ID, item.ID, trial.ID, observation.ReplicateNo, observation.DayNo, observation.Revision, observationData, observation.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
			if err != nil {
				return fmt.Errorf("append supplemental observation revision: %w", err)
			}
		}
		for snapshotIndex := range trial.AnalysisSnapshots {
			snapshot := &trial.AnalysisSnapshots[snapshotIndex]
			snapshotData, err := encode(snapshot)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO supplemental_analysis_snapshots(id, case_id, supplemental_trial_id, sequence, snapshot_json, calculated_at) VALUES(?, ?, ?, ?, ?, ?)`, snapshot.ID, item.ID, trial.ID, snapshot.Sequence, snapshotData, snapshot.CalculatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
			if err != nil {
				return fmt.Errorf("append supplemental analysis snapshot: %w", err)
			}
		}
	}
	for index := range item.AnalysisSnapshots {
		snapshot := &item.AnalysisSnapshots[index]
		data, err := encode(snapshot)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO analysis_snapshots(id, case_id, sequence, snapshot_json, calculated_at) VALUES(?, ?, ?, ?, ?)`, snapshot.ID, item.ID, snapshot.Sequence, data, snapshot.CalculatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
		if err != nil {
			return fmt.Errorf("append analysis snapshot: %w", err)
		}
	}
	if item.EvidenceBundle != nil {
		data, err := encode(item.EvidenceBundle)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO evidence_bundles(id, case_id, digest, bundle_json, frozen_at) VALUES(?, ?, ?, ?, ?)`, item.EvidenceBundle.ID, item.ID, item.EvidenceBundle.Digest, data, item.EvidenceBundle.FrozenAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
		if err != nil {
			return fmt.Errorf("freeze evidence bundle: %w", err)
		}
	}
	if item.Credential != nil {
		data, err := encode(item.Credential)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO credentials(credential_no, case_id, evidence_bundle_id, evidence_digest, credential_json, issued_at) VALUES(?, ?, ?, ?, ?, ?)`, item.Credential.CredentialNo, item.ID, item.Credential.EvidenceBundleID, item.Credential.EvidenceDigest, data, item.Credential.IssuedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))
		if err != nil {
			return fmt.Errorf("issue immutable credential: %w", err)
		}
	}
	return nil
}
