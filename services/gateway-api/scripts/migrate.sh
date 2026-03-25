#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
DB_HOST="${PG_HOST:-localhost}"
DB_PORT="${PG_PORT:-5432}"
DB_USER="${PG_USER:-postgres}"
DB_PASSWORD="${PG_PASSWORD:-postgres}"
DB_NAME="${PG_DATABASE:-viola_gateway}"
MIGRATIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/migrations"

# Print usage
usage() {
    cat <<EOF
Usage: $0 [OPTIONS] COMMAND

Database migration tool for gateway-api

COMMANDS:
    up              Apply all pending migrations
    down            Rollback the last migration
    status          Show migration status
    create NAME     Create a new migration file
    force VERSION   Mark a migration as applied without running it

OPTIONS:
    -h, --host      Database host (default: localhost)
    -p, --port      Database port (default: 5432)
    -u, --user      Database user (default: postgres)
    -d, --database  Database name (default: viola_gateway)
    --help          Show this help message

ENVIRONMENT VARIABLES:
    PG_HOST, PG_PORT, PG_USER, PG_PASSWORD, PG_DATABASE

EXAMPLES:
    # Apply all migrations
    $0 up

    # Check migration status
    $0 status

    # Create new migration
    $0 create add_user_table

    # Rollback last migration
    $0 down

EOF
}

# Print colored message
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Execute SQL query
psql_exec() {
    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t -A -c "$1"
}

# Execute SQL file
psql_file() {
    PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "$1"
}

# Check if database exists
check_database() {
    if ! PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -lqt | cut -d \| -f 1 | grep -qw "${DB_NAME}"; then
        log_error "Database '${DB_NAME}' does not exist"
        log_info "Create it with: createdb -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} ${DB_NAME}"
        exit 1
    fi
}

# Initialize migration tracking table
init_migrations_table() {
    psql_exec "CREATE TABLE IF NOT EXISTS schema_migrations (
        version TEXT PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );" > /dev/null
    log_info "Migration tracking table initialized"
}

# Get list of applied migrations
get_applied_migrations() {
    psql_exec "SELECT version FROM schema_migrations ORDER BY version;" 2>/dev/null || echo ""
}

# Get list of migration files
get_migration_files() {
    find "${MIGRATIONS_DIR}" -name "*.sql" -type f | sort
}

# Apply a single migration
apply_migration() {
    local migration_file=$1
    local version=$(basename "${migration_file}" .sql)

    log_info "Applying migration: ${version}"

    if ! psql_file "${migration_file}"; then
        log_error "Failed to apply migration: ${version}"
        exit 1
    fi

    psql_exec "INSERT INTO schema_migrations (version) VALUES ('${version}');" > /dev/null
    log_info "✓ Applied: ${version}"
}

# Rollback a single migration
rollback_migration() {
    local version=$1
    local migration_file="${MIGRATIONS_DIR}/${version}.sql"

    # Check if rollback file exists
    local rollback_file="${MIGRATIONS_DIR}/${version}.down.sql"
    if [[ ! -f "${rollback_file}" ]]; then
        log_error "No rollback file found: ${rollback_file}"
        log_warn "Manual rollback required for: ${version}"
        exit 1
    fi

    log_info "Rolling back migration: ${version}"

    if ! psql_file "${rollback_file}"; then
        log_error "Failed to rollback migration: ${version}"
        exit 1
    fi

    psql_exec "DELETE FROM schema_migrations WHERE version = '${version}';" > /dev/null
    log_info "✓ Rolled back: ${version}"
}

# Command: up
cmd_up() {
    check_database
    init_migrations_table

    local applied_migrations=$(get_applied_migrations)
    local pending_count=0

    for migration_file in $(get_migration_files); do
        local version=$(basename "${migration_file}" .sql)

        # Skip if already applied
        if echo "${applied_migrations}" | grep -q "^${version}$"; then
            continue
        fi

        apply_migration "${migration_file}"
        ((pending_count++))
    done

    if [[ ${pending_count} -eq 0 ]]; then
        log_info "No pending migrations"
    else
        log_info "Applied ${pending_count} migration(s)"
    fi
}

# Command: down
cmd_down() {
    check_database
    init_migrations_table

    local applied_migrations=$(get_applied_migrations)
    local last_migration=$(echo "${applied_migrations}" | tail -n 1)

    if [[ -z "${last_migration}" ]]; then
        log_warn "No migrations to rollback"
        exit 0
    fi

    rollback_migration "${last_migration}"
}

# Command: status
cmd_status() {
    check_database
    init_migrations_table

    local applied_migrations=$(get_applied_migrations)

    echo ""
    echo "Migration Status"
    echo "================"
    echo ""

    for migration_file in $(get_migration_files); do
        local version=$(basename "${migration_file}" .sql)

        if echo "${applied_migrations}" | grep -q "^${version}$"; then
            echo -e "${GREEN}✓${NC} ${version} (applied)"
        else
            echo -e "${YELLOW}○${NC} ${version} (pending)"
        fi
    done

    echo ""
}

# Command: create
cmd_create() {
    local name=$1

    if [[ -z "${name}" ]]; then
        log_error "Migration name required"
        echo "Usage: $0 create NAME"
        exit 1
    fi

    # Get next version number
    local last_file=$(find "${MIGRATIONS_DIR}" -name "*.sql" -type f | sort | tail -n 1)
    local next_version=0001

    if [[ -n "${last_file}" ]]; then
        local last_version=$(basename "${last_file}" .sql | cut -d_ -f1)
        next_version=$(printf "%04d" $((10#${last_version} + 1)))
    fi

    local migration_name="${next_version}_${name}"
    local up_file="${MIGRATIONS_DIR}/${migration_name}.sql"
    local down_file="${MIGRATIONS_DIR}/${migration_name}.down.sql"

    # Create up migration
    cat > "${up_file}" <<EOF
-- Migration: ${migration_name}
-- Created: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

-- Add your migration SQL here

EOF

    # Create down migration
    cat > "${down_file}" <<EOF
-- Rollback: ${migration_name}
-- Created: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

-- Add your rollback SQL here

EOF

    log_info "Created migration files:"
    echo "  ${up_file}"
    echo "  ${down_file}"
}

# Command: force
cmd_force() {
    local version=$1

    if [[ -z "${version}" ]]; then
        log_error "Version required"
        echo "Usage: $0 force VERSION"
        exit 1
    fi

    check_database
    init_migrations_table

    psql_exec "INSERT INTO schema_migrations (version) VALUES ('${version}') ON CONFLICT DO NOTHING;" > /dev/null
    log_info "Marked as applied: ${version}"
}

# Parse arguments
COMMAND=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            DB_HOST="$2"
            shift 2
            ;;
        -p|--port)
            DB_PORT="$2"
            shift 2
            ;;
        -u|--user)
            DB_USER="$2"
            shift 2
            ;;
        -d|--database)
            DB_NAME="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        up|down|status|create|force)
            COMMAND="$1"
            shift
            break
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check command
if [[ -z "${COMMAND}" ]]; then
    log_error "Command required"
    usage
    exit 1
fi

# Execute command
case "${COMMAND}" in
    up)
        cmd_up
        ;;
    down)
        cmd_down
        ;;
    status)
        cmd_status
        ;;
    create)
        cmd_create "$@"
        ;;
    force)
        cmd_force "$@"
        ;;
    *)
        log_error "Unknown command: ${COMMAND}"
        usage
        exit 1
        ;;
esac
