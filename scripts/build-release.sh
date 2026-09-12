#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "exactly one reviewed release tag is required" >&2
  exit 2
fi
tag="$1"
export GOMAXPROCS="${GOMAXPROCS:-1}"
go run -p 1 ./cmd/releasecheck --tag "$tag" > /dev/null

version="${tag#v}"
revision="$(git rev-parse HEAD)"
test "$(pwd -P)" = "$(git rev-parse --show-toplevel)"
source_status="$(git status --porcelain --untracked-files=normal)"
test -z "$source_status"
test "$(git rev-parse --verify "refs/tags/${tag}^{commit}")" = "$revision"
umask 077
# A pre-existing path belongs to its owner. Never clear it or reuse partial work.
mkdir dist

targets=(
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"$target"
  name="statetwin_${tag}_${goos}_${goarch}"
  suffix=""
  if [[ "$goos" == "windows" ]]; then
    suffix=".exe"
  fi
  binary="dist/${name}${suffix}"
  echo "building ${binary}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -p 1 \
    -trimpath -ldflags "-s -w -X github.com/augety121/mcp-state-twin/internal/server.Version=${version} -X github.com/augety121/mcp-state-twin/internal/server.Revision=${revision}" \
    -o "$binary" ./cmd/statetwin
done

(cd dist && sha256sum * > SHA256SUMS)
printf '%s\n' "MCP State Twin ${tag}" > dist/RELEASE.txt
