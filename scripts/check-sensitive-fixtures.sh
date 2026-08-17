#!/usr/bin/env bash
set -euo pipefail

scope=(examples testdata)
patterns=(
  '-----BEGIN ([A-Z0-9 ]+ )?PRIVATE KEY-----'
  '(sk|rk)-[A-Za-z0-9_-]{20,}'
  'gh[pousr]_[A-Za-z0-9]{20,}'
  'AIza[0-9A-Za-z_-]{30,}'
  '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
  'Authorization:[[:space:]]*(Bearer|Basic)[[:space:]]+'
)

for pattern in "${patterns[@]}"; do
  set +e
  git grep -n -I -E -e "$pattern" -- "${scope[@]}"
  status=$?
  set -e
  if [[ $status -eq 0 ]]; then
    echo "sensitive fixture policy rejected the match above" >&2
    exit 1
  fi
  if [[ $status -ne 1 ]]; then
    echo "sensitive fixture policy scanner failed with exit code $status" >&2
    exit "$status"
  fi
done

echo "synthetic fixture policy passed"
