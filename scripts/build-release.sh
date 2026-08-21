#!/usr/bin/env bash
set -euo pipefail

tag="${1:-}"
if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid release tag: $tag" >&2
  exit 2
fi

version="${tag#v}"
rm -rf dist
mkdir -p dist

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
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -trimpath -ldflags "-s -w -X github.com/augety121/mcp-state-twin/internal/server.Version=${version}" \
    -o "$binary" ./cmd/statetwin
done

(cd dist && sha256sum * > SHA256SUMS)
printf '%s\n' "MCP State Twin ${tag}" > dist/RELEASE.txt
