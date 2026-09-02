#!/usr/bin/env bash
# SPEC-0035: count-bounded smoke; deadlines remain failures, never retries.
set -euo pipefail

case "${1:-}" in
  spec) package=./internal/spec; target=FuzzDecodeTwinSpec; iterations=200000x ;;
  engine) package=./internal/engine; target=FuzzExpressionCompilation; iterations=10000x ;;
  *) echo 'usage: run-bounded-fuzz.sh spec|engine' >&2; exit 2 ;;
esac

export GOMAXPROCS=1
go version
printf 'Fuzz profile: target=%s iterations=%s workers=1 timeout=3m minimization=1000x\n' "$target" "$iterations"
go test -p 1 "$package" -run='^$' -fuzz="^${target}$" \
  -fuzztime="$iterations" -fuzzminimizetime=1000x -parallel=1 -timeout=3m
