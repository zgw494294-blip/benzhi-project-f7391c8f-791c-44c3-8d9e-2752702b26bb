package store

const schemaVersion = 3

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);`,
	`CREATE TABLE IF NOT EXISTS cases (
 id TEXT PRIMARY KEY, accession_code TEXT NOT NULL, status TEXT NOT NULL, version INTEGER NOT NULL,
 aggregate_json BLOB NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);`,
	`CREATE INDEX IF NOT EXISTS idx_cases_status_updated ON cases(status, updated_at DESC);`,
	`CREATE TABLE IF NOT EXISTS observation_revisions (
 id TEXT NOT NULL, case_id TEXT NOT NULL, replicate_no INTEGER NOT NULL, day_no INTEGER NOT NULL,
 revision INTEGER NOT NULL, evidence_json BLOB NOT NULL, recorded_at TEXT NOT NULL,
 PRIMARY KEY(case_id, replicate_no, day_no, revision), FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TABLE IF NOT EXISTS deviations (
 id TEXT PRIMARY KEY, case_id TEXT NOT NULL, rule_code TEXT NOT NULL, status TEXT NOT NULL,
 evidence_json BLOB NOT NULL, FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TABLE IF NOT EXISTS supplemental_trials (
 id TEXT PRIMARY KEY, case_id TEXT NOT NULL, deviation_id TEXT NOT NULL UNIQUE, status TEXT NOT NULL,
 trial_json BLOB NOT NULL, initiated_at TEXT NOT NULL, FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TABLE IF NOT EXISTS supplemental_observation_revisions (
 id TEXT NOT NULL, case_id TEXT NOT NULL, supplemental_trial_id TEXT NOT NULL, replicate_no INTEGER NOT NULL,
 day_no INTEGER NOT NULL, revision INTEGER NOT NULL, evidence_json BLOB NOT NULL, recorded_at TEXT NOT NULL,
 PRIMARY KEY(supplemental_trial_id, replicate_no, day_no, revision),
 FOREIGN KEY(case_id) REFERENCES cases(id), FOREIGN KEY(supplemental_trial_id) REFERENCES supplemental_trials(id)
);`,
	`CREATE TABLE IF NOT EXISTS supplemental_analysis_snapshots (
 id TEXT PRIMARY KEY, case_id TEXT NOT NULL, supplemental_trial_id TEXT NOT NULL, sequence INTEGER NOT NULL,
 snapshot_json BLOB NOT NULL, calculated_at TEXT NOT NULL, UNIQUE(supplemental_trial_id, sequence),
 FOREIGN KEY(case_id) REFERENCES cases(id), FOREIGN KEY(supplemental_trial_id) REFERENCES supplemental_trials(id)
);`,
	`CREATE TABLE IF NOT EXISTS analysis_snapshots (
 id TEXT PRIMARY KEY, case_id TEXT NOT NULL, sequence INTEGER NOT NULL, snapshot_json BLOB NOT NULL,
 calculated_at TEXT NOT NULL, UNIQUE(case_id, sequence), FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TABLE IF NOT EXISTS evidence_bundles (
 id TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE, digest TEXT NOT NULL UNIQUE, bundle_json BLOB NOT NULL,
 frozen_at TEXT NOT NULL, FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TABLE IF NOT EXISTS credentials (
 credential_no TEXT PRIMARY KEY, case_id TEXT NOT NULL UNIQUE, evidence_bundle_id TEXT NOT NULL UNIQUE,
 evidence_digest TEXT NOT NULL, credential_json BLOB NOT NULL, issued_at TEXT NOT NULL,
 FOREIGN KEY(case_id) REFERENCES cases(id), FOREIGN KEY(evidence_bundle_id) REFERENCES evidence_bundles(id)
);`,
	`CREATE TABLE IF NOT EXISTS idempotency_results (
 idempotency_key TEXT PRIMARY KEY, action TEXT NOT NULL, case_id TEXT NOT NULL,
 response_json BLOB NOT NULL, request_digest TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
);`,
	`CREATE TABLE IF NOT EXISTS audit_events (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT, case_id TEXT NOT NULL, action TEXT NOT NULL,
 actor TEXT NOT NULL, details_json BLOB NOT NULL, occurred_at TEXT NOT NULL, FOREIGN KEY(case_id) REFERENCES cases(id)
);`,
	`CREATE TRIGGER IF NOT EXISTS evidence_bundles_no_update BEFORE UPDATE ON evidence_bundles BEGIN SELECT RAISE(ABORT, 'evidence bundle is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS evidence_bundles_no_delete BEFORE DELETE ON evidence_bundles BEGIN SELECT RAISE(ABORT, 'evidence bundle is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS credentials_no_update BEFORE UPDATE ON credentials BEGIN SELECT RAISE(ABORT, 'credential is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS credentials_no_delete BEFORE DELETE ON credentials BEGIN SELECT RAISE(ABORT, 'credential is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS observation_revisions_no_update BEFORE UPDATE ON observation_revisions BEGIN SELECT RAISE(ABORT, 'observation revision is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS observation_revisions_no_delete BEFORE DELETE ON observation_revisions BEGIN SELECT RAISE(ABORT, 'observation revision is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS analysis_snapshots_no_update BEFORE UPDATE ON analysis_snapshots BEGIN SELECT RAISE(ABORT, 'analysis snapshot is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS analysis_snapshots_no_delete BEFORE DELETE ON analysis_snapshots BEGIN SELECT RAISE(ABORT, 'analysis snapshot is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS supplemental_observations_no_update BEFORE UPDATE ON supplemental_observation_revisions BEGIN SELECT RAISE(ABORT, 'supplemental observation revision is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS supplemental_observations_no_delete BEFORE DELETE ON supplemental_observation_revisions BEGIN SELECT RAISE(ABORT, 'supplemental observation revision is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS supplemental_analysis_no_update BEFORE UPDATE ON supplemental_analysis_snapshots BEGIN SELECT RAISE(ABORT, 'supplemental analysis snapshot is immutable'); END;`,
	`CREATE TRIGGER IF NOT EXISTS supplemental_analysis_no_delete BEFORE DELETE ON supplemental_analysis_snapshots BEGIN SELECT RAISE(ABORT, 'supplemental analysis snapshot is immutable'); END;`,
}
