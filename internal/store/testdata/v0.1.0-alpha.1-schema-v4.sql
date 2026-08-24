-- Source tag: v0.1.0-alpha.1
-- Source commit: 8eba96819d865a0616b425751a9ca2cb2d780c54
-- Storage schema: 4
-- Synthetic fixture only. It contains no production data or credentials.
PRAGMA application_id = 1398036302;
PRAGMA user_version = 4;

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

CREATE TABLE fault_plans (
  id TEXT NOT NULL,
  branch_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  phase TEXT NOT NULL,
  error_class TEXT NOT NULL,
  message TEXT NOT NULL,
  remaining_count INTEGER NOT NULL CHECK(remaining_count >= 0 AND remaining_count <= 1000),
  fired_count INTEGER NOT NULL DEFAULT 0 CHECK(fired_count >= 0 AND fired_count <= 1000),
  created_at TEXT NOT NULL,
  CHECK(remaining_count + fired_count >= 1 AND remaining_count + fired_count <= 1000),
  CHECK(phase IN ('before-validation', 'after-commit-before-response')),
  CHECK(error_class IN ('RATE_LIMITED', 'TIMEOUT_BEFORE_EFFECT', 'TIMEOUT_AFTER_EFFECT')),
  CHECK((phase = 'after-commit-before-response' AND error_class = 'TIMEOUT_AFTER_EFFECT') OR
        (phase = 'before-validation' AND error_class IN ('RATE_LIMITED', 'TIMEOUT_BEFORE_EFFECT'))),
  PRIMARY KEY(branch_id, id),
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

CREATE TABLE fault_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  branch_id TEXT NOT NULL,
  fault_id TEXT NOT NULL,
  call_index INTEGER NOT NULL,
  phase TEXT NOT NULL,
  error_class TEXT NOT NULL,
  before_digest TEXT NOT NULL,
  after_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);

INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count, head_version)
VALUES(
  'main',
  'sha256:tagged-spec',
  '{"entities":{"item":{"legacy":{"id":"legacy","state":"open"}}},"sequences":{}}',
  'sha256:tagged-state',
  '2026-08-21T00:00:00Z',
  1,
  2
);

INSERT INTO snapshots(id, name, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version, created_at)
VALUES(
  'tagged-snapshot-id',
  'tagged-base',
  'sha256:tagged-spec',
  '{"entities":{"item":{"legacy":{"id":"legacy","state":"open"}}},"sequences":{}}',
  'sha256:tagged-state',
  '2026-08-21T00:00:00Z',
  2,
  4,
  '2026-08-21T00:00:00Z'
);

INSERT INTO audit(id, branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at)
VALUES(1, 'main', 1, 'get_item', '{}', '{"id":"legacy"}', '', 'sha256:tagged-state', 'sha256:tagged-state', '2026-08-21T00:00:01Z');

INSERT INTO control_audit(id, operation, branch_id, snapshot_name, before_digest, after_digest, created_at)
VALUES(1, 'snapshot.create', 'main', 'tagged-base', 'sha256:tagged-state', 'sha256:tagged-state', '2026-08-21T00:00:02Z');

INSERT INTO fault_plans(id, branch_id, tool_name, phase, error_class, message, remaining_count, fired_count, created_at)
VALUES('tagged-rate-limit', 'main', 'get_item', 'before-validation', 'RATE_LIMITED', 'synthetic tagged fixture', 1, 0, '2026-08-21T00:00:03Z');
