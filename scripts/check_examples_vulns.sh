#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLES_DIR="$ROOT_DIR/examples"

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but was not found in PATH" >&2
  exit 1
fi

GOVULNCHECK_BIN=""
if command -v govulncheck >/dev/null 2>&1; then
  GOVULNCHECK_BIN="govulncheck"
else
  echo "govulncheck not found; installing..."
  GOBIN="${GOBIN:-"$(go env GOPATH)/bin"}"
  PATH="$GOBIN:$PATH"
  go install golang.org/x/vuln/cmd/govulncheck@latest
  GOVULNCHECK_BIN="$GOBIN/govulncheck"
fi

mapfile -d '' MOD_FILES < <(find "$EXAMPLES_DIR" -name go.mod -print0)
if [ "${#MOD_FILES[@]}" -eq 0 ]; then
  echo "No go.mod files found under $EXAMPLES_DIR"
  exit 0
fi

for MOD_FILE in "${MOD_FILES[@]}"; do
  MOD_DIR="$(dirname "$MOD_FILE")"
  echo "==> Checking $MOD_DIR"
  pushd "$MOD_DIR" >/dev/null

  if "$GOVULNCHECK_BIN" ./...; then
    echo "No vulnerabilities found."
    popd >/dev/null
    continue
  fi

  STATUS=$?
  if [ "$STATUS" -ne 1 ]; then
    echo "govulncheck failed with status $STATUS" >&2
    exit "$STATUS"
  fi

  echo "Vulnerabilities found; updating dependencies..."
  go get -u ./...
  go mod tidy

  "$GOVULNCHECK_BIN" ./... || true

  popd >/dev/null
done
