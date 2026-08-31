-- Source commit: b8e836bd5df2442b4799f296a7e68047595532cd
-- Synthetic Episode Journal schema v1 fixture. No production data or secrets.
PRAGMA application_id = 1162889804;
PRAGMA user_version = 1;

CREATE TABLE episodes (
  id TEXT PRIMARY KEY,
  request_digest TEXT NOT NULL,
  bundle_digest TEXT NOT NULL,
  scenario_path TEXT NOT NULL,
  runtime_version TEXT NOT NULL,
  runtime_revision TEXT NOT NULL,
  status TEXT NOT NULL,
  sequence INTEGER NOT NULL CHECK(sequence >= 0),
  outcome TEXT NOT NULL DEFAULT '',
  evidence_json BLOB,
  evidence_digest TEXT NOT NULL DEFAULT '',
  error_class TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(status IN ('CREATED','PROVISIONING','READY','RUNNING','EVALUATING','SUCCEEDED','ASSERTION_FAILED','RUNTIME_ERROR','CANCELLED'))
);
CREATE TABLE episode_events (
  episode_id TEXT NOT NULL,
  sequence INTEGER NOT NULL CHECK(sequence >= 0),
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(episode_id, sequence),
  FOREIGN KEY(episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);

INSERT INTO episodes(
  id, request_digest, bundle_digest, scenario_path, runtime_version,
  runtime_revision, status, sequence, created_at, updated_at
) VALUES(
  'published-v1',
  'sha256:163a58a7c15adc635d51bd3f63bb7038085f2ddb8893d6e7a8b6be6cb71cb927',
  'sha256:fixture',
  'scenarios/test.yaml',
  '0.1.0-dev',
  'b8e836bd5df2442b4799f296a7e68047595532cd',
  'CREATED',
  0,
  '2026-08-26T13:32:11Z',
  '2026-08-26T13:32:11Z'
);
INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at)
VALUES('published-v1', 0, '', 'CREATED', '2026-08-26T13:32:11Z');
