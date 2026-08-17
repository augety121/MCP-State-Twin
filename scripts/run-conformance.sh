#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./statetwin}"
port="${STATETWIN_CONFORMANCE_PORT:-18090}"
control_port="${STATETWIN_CONFORMANCE_CONTROL_PORT:-18091}"
database="$(mktemp "${TMPDIR:-/tmp}/statetwin-conformance.XXXXXX.db")"
export STATETWIN_CONTROL_TOKEN="synthetic-conformance-token"

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi
  rm -f -- "${database}" "${database}-shm" "${database}-wal"
}
trap cleanup EXIT

"${binary}" serve \
  --spec testdata/conformance/twin.yaml \
  --fixture testdata/conformance/state.json \
  --db "${database}" \
  --data-addr "127.0.0.1:${port}" \
  --control-addr "127.0.0.1:${control_port}" &
server_pid=$!

for _ in $(seq 1 40); do
  if curl --silent --output /dev/null --request POST "http://127.0.0.1:${port}/mcp/main"; then
    break
  fi
  sleep 0.25
done
if ! kill -0 "${server_pid}" 2>/dev/null; then
  echo "State Twin conformance server exited before tests" >&2
  exit 1
fi

for scenario in server-initialize ping tools-list json-schema-2020-12; do
  npx --yes @modelcontextprotocol/conformance@0.1.16 server \
    --url "http://127.0.0.1:${port}/mcp/main" \
    --scenario "${scenario}"
done
