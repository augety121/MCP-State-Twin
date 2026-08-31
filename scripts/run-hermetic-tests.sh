#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./statetwin}"
interfaces="$(ip -o link show | awk -F': ' '{name=$2; sub(/@.*/, "", name); print name}' | sort)"
if [[ "$interfaces" != "lo" ]]; then
  echo "hermetic test requires a network namespace containing only loopback; found: $interfaces" >&2
  exit 1
fi

ip link set lo up
export GOPROXY=off
export GOSUMDB=off

go test ./...

workdir="$(mktemp -d)"
coordinator_pid=""
cleanup() {
  if [[ -n "$coordinator_pid" ]]; then
    kill "$coordinator_pid" 2>/dev/null || true
    wait "$coordinator_pid" 2>/dev/null || true
  fi
  rm -rf -- "$workdir"
}
trap cleanup EXIT

"$binary" validate --spec examples/issue-tracker/twin.yaml
"$binary" init \
  --spec examples/issue-tracker/twin.yaml \
  --fixture examples/issue-tracker/state.json \
  --db "$workdir/hermetic.db" \
  --snapshot base
"$binary" fork --db "$workdir/hermetic.db" --snapshot base --branch isolated
"$binary" call \
  --spec examples/issue-tracker/twin.yaml \
  --db "$workdir/hermetic.db" \
  --branch isolated \
  --tool get_issue \
  --input '{"owner":"octo","repository":"demo","number":1}'
"$binary" bundle build \
  --manifest examples/issue-tracker/bundle.yaml \
  --out "$workdir/issue-tracker.stb"
"$binary" bundle verify --bundle "$workdir/issue-tracker.stb"
"$binary" episode run \
  --bundle "$workdir/issue-tracker.stb" \
  --id hermetic-episode \
  --out "$workdir/episode-evidence.json"
"$binary" episode run \
  --bundle "$workdir/issue-tracker.stb" \
  --id hermetic-durable \
  --journal "$workdir/episodes.db"
"$binary" episode run \
  --bundle "$workdir/issue-tracker.stb" \
  --id hermetic-durable \
  --journal "$workdir/episodes.db"
"$binary" episode inspect \
  --journal "$workdir/episodes.db" \
  --id hermetic-durable

"$binary" episode submit \
  --bundle "$workdir/issue-tracker.stb" \
  --id hermetic-remote \
  --journal "$workdir/remote-episodes.db" \
  --effect-profile hermetic \
  --max-attempts 2
export STATETWIN_COORDINATOR_TOKEN="synthetic-hermetic-token"
"$binary" episode coordinator \
  --journal "$workdir/remote-episodes.db" \
  --addr 127.0.0.1:18092 \
  >"$workdir/coordinator.log" 2>&1 &
coordinator_pid="$!"
for _ in {1..50}; do
  if curl --fail --silent --show-error \
    --header "Authorization: Bearer ${STATETWIN_COORDINATOR_TOKEN}" \
    "http://127.0.0.1:18092/v1/episodes/hermetic-remote" \
    >"$workdir/task-before.json"; then
    break
  fi
  sleep 0.1
done
if ! kill -0 "$coordinator_pid" 2>/dev/null; then
  cat "$workdir/coordinator.log" >&2
  exit 1
fi
"$binary" episode worker \
  --coordinator http://127.0.0.1:18092 \
  --id hermetic-worker \
  --once
"$binary" episode task \
  --journal "$workdir/remote-episodes.db" \
  --id hermetic-remote \
  >"$workdir/task-after.json"
grep -q '"state": "COMPLETED"' "$workdir/task-after.json"
