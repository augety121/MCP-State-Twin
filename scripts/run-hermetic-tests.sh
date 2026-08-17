#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./statetwin}"
interfaces="$(find /sys/class/net -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)"
if [[ "$interfaces" != "lo" ]]; then
  echo "hermetic test requires a network namespace containing only loopback; found: $interfaces" >&2
  exit 1
fi

ip link set lo up
export GOPROXY=off
export GOSUMDB=off

go test ./...

workdir="$(mktemp -d)"
trap 'rm -rf -- "$workdir"' EXIT

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
