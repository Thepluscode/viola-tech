#!/usr/bin/env bash
set -euo pipefail

docker compose -f tests/integration/docker-compose.yaml up -d
