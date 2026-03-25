#!/usr/bin/env bash
set -euo pipefail

if ! command -v buf >/dev/null 2>&1; then
  echo "ERROR: buf is not installed. Install from https://buf.build/docs/installation"
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

buf lint
buf generate

echo "Generated Go protos under shared/go/proto"
