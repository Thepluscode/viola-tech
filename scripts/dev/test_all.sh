#!/usr/bin/env bash
set -euo pipefail

# Shared module
(cd shared/go && go test ./...)

# Agent + services
(cd agent && go test ./...)
for svc in services/*; do
  if [ -f "$svc/go.mod" ]; then
    (cd "$svc" && go test ./...)
  fi
done
