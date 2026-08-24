-- Source commit: 337ccf0 (schema v3 before fault tables were introduced)
-- Synthetic fixture only. It contains no production data or credentials.
PRAGMA application_id = 1398036302;
PRAGMA user_version = 3;

CREATE TABLE branches (
  id TEXT PRIMARY KEY,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0,
  head_version INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE snapshots (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  source_head_version INTEGER NOT NULL DEFAULT 0,
  storage_schema_version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);

CREATE TABLE audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  branch_id TEXT NOT NULL,
  call_index INTEGER NOT NULL,
  tool_name TEXT NOT NULL,
  input_json BLOB NOT NULL,
  result_json BLOB NOT NULL,
  error_class TEXT NOT NULL,
  before_digest TEXT NOT NULL,
  after_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

CREATE TABLE control_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  operation TEXT NOT NULL,
  branch_id TEXT NOT NULL DEFAULT '',
  snapshot_name TEXT NOT NULL DEFAULT '',
  before_digest TEXT NOT NULL DEFAULT '',
  after_digest TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count, head_version)
VALUES(
  'main',
  'sha256:v3-spec',
  '{"entities":{"item":{"legacy":{"id":"legacy","state":"open"}}},"sequences":{}}',
  'sha256:v3-state',
  '2026-08-17T00:00:00Z',
  3,
  5
);

INSERT INTO snapshots(id, name, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version, created_at)
VALUES(
  'v3-snapshot-id',
  'v3-base',
  'sha256:v3-spec',
  '{"entities":{"item":{"legacy":{"id":"legacy","state":"open"}}},"sequences":{}}',
  'sha256:v3-state',
  '2026-08-17T00:00:00Z',
  5,
  3,
  '2026-08-17T00:00:00Z'
);

INSERT INTO audit(id, branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at)
VALUES(1, 'main', 3, 'get_item', '{}', '{"id":"legacy"}', '', 'sha256:v3-state', 'sha256:v3-state', '2026-08-17T00:00:01Z');

INSERT INTO control_audit(id, operation, branch_id, snapshot_name, before_digest, after_digest, created_at)
VALUES(1, 'snapshot.create', 'main', 'v3-base', 'sha256:v3-state', 'sha256:v3-state', '2026-08-17T00:00:02Z');
