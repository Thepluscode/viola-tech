.PHONY: help dev dev-infra dev-monitoring down smoke healthcheck demo migrate protos lint test build test-integration test-rbac test-e2e

help:
	@echo ""
	@echo "  Viola XDR — Development Commands"
	@echo ""
	@echo "  make dev              Start full stack (infra + all services)"
	@echo "  make dev-infra        Start infra only (postgres + kafka)"
	@echo "  make dev-monitoring   Start full stack + Prometheus/Grafana"
	@echo "  make down             Stop and remove all containers"
	@echo "  make demo             Seed demo data through the full pipeline"
	@echo "  make smoke            Run end-to-end smoke test (pipeline)"
	@echo "  make healthcheck      Run quick service health checks"
	@echo "  make migrate          Apply database migrations"
	@echo "  make migrate-down     Rollback last migration"
	@echo "  make migrate-version  Show current migration version"
	@echo "  make build            Build all service binaries"
	@echo "  make test             Run all unit tests"
	@echo "  make test-integration Run integration tests (requires running stack)"
	@echo "  make test-rbac        Run RBAC + tenant isolation tests"
	@echo "  make test-e2e         Run Playwright E2E tests (frontend)"
	@echo "  make protos           Regenerate protobuf Go code"
	@echo "  make lint             Lint all services"
	@echo ""

dev:
	docker compose up --build

dev-infra:
	docker compose up postgres kafka

dev-monitoring:
	docker compose -f docker-compose.yml -f ops/monitoring/docker-compose.monitoring.yml up --build

down:
	docker compose down -v

demo:
	@echo "Seeding demo data..."
	go run ./scripts/dev/demo/

smoke:
	@echo "Running smoke test..."
	go run ./scripts/dev/smoke/

healthcheck:
	@go run ./scripts/dev/healthcheck/ --verbose

migrate:
	@go run ./scripts/dev/migrate/ up

migrate-down:
	@go run ./scripts/dev/migrate/ down

migrate-version:
	@go run ./scripts/dev/migrate/ version

protos:
	@bash scripts/dev/gen_protos.sh

lint:
	@bash scripts/dev/lint_all.sh

test:
	@bash scripts/dev/test_all.sh

test-integration:
	@echo "Running integration tests..."
	cd tests/integration && go test -v -count=1 -timeout 120s ./...

test-rbac:
	@echo "Running RBAC + tenant isolation tests..."
	cd tests/integration && go test -v -run TestRBAC -count=1 -timeout 60s ./...

test-e2e:
	@cd ui/web && npx playwright test

build:
	@for svc in gateway-api detection workers ingestion graph auth intel response cloud-connector; do \
		echo "Building $$svc..."; \
		(cd services/$$svc && go build ./...); \
	done
	@(cd agent && go build ./...) && echo "Building agent..."
	@echo "All services built."
