// migrate/main.go — Viola XDR database migration runner
//
// Usage:
//   go run ./scripts/dev/migrate/ up        # Apply all pending migrations
//   go run ./scripts/dev/migrate/ down      # Rollback last migration
//   go run ./scripts/dev/migrate/ version   # Show current version
//   go run ./scripts/dev/migrate/ force N   # Force set version to N
//
// Env:
//   DATABASE_URL (default: postgres://viola:viola@localhost:5435/viola?sslmode=disable)
//   MIGRATIONS_PATH (default: migrations/)
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultDSN = "postgres://viola:viola@localhost:5435/viola?sslmode=disable"

func main() {
	dsn := getenv("DATABASE_URL", defaultDSN)
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations/")

	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate <up|down|version|force N>")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatal("connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fatal("ping: %v", err)
	}

	// Ensure schema_migrations table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INT  NOT NULL,
			dirty      BOOL NOT NULL DEFAULT false,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (version)
		)
	`)
	if err != nil {
		fatal("create schema_migrations: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		if err := migrateUp(db, migrationsPath); err != nil {
			fatal("up: %v", err)
		}
	case "down":
		if err := migrateDown(db, migrationsPath); err != nil {
			fatal("down: %v", err)
		}
	case "version":
		v, dirty := currentVersion(db)
		dirtyStr := ""
		if dirty {
			dirtyStr = " (dirty)"
		}
		fmt.Printf("Current version: %d%s\n", v, dirtyStr)
	case "force":
		if len(os.Args) < 3 {
			fatal("usage: migrate force N")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fatal("invalid version: %v", err)
		}
		if err := forceVersion(db, v); err != nil {
			fatal("force: %v", err)
		}
		fmt.Printf("Forced version to %d\n", v)
	default:
		fatal("unknown command: %s", cmd)
	}
}

type migration struct {
	version int
	name    string
	upFile  string
	downFile string
}

func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	mmap := make(map[int]*migration)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse: 000001_name.up.sql or 000001_name.down.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		m, ok := mmap[v]
		if !ok {
			m = &migration{version: v}
			mmap[v] = m
		}

		path := filepath.Join(dir, name)
		if strings.Contains(name, ".up.sql") {
			m.upFile = path
			m.name = strings.TrimSuffix(parts[1], ".up.sql")
		} else if strings.Contains(name, ".down.sql") {
			m.downFile = path
		}
	}

	var migrations []migration
	for _, m := range mmap {
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

func currentVersion(db *sql.DB) (int, bool) {
	var v int
	var dirty bool
	err := db.QueryRow(`SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&v, &dirty)
	if err != nil {
		return 0, false
	}
	return v, dirty
}

func migrateUp(db *sql.DB, dir string) error {
	migrations, err := loadMigrations(dir)
	if err != nil {
		return err
	}

	current, dirty := currentVersion(db)
	if dirty {
		return fmt.Errorf("database is dirty at version %d — run 'force' to fix", current)
	}

	applied := 0
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if m.upFile == "" {
			return fmt.Errorf("no up file for version %d", m.version)
		}

		content, err := os.ReadFile(m.upFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.upFile, err)
		}

		// Mark as dirty before applying
		_, err = db.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES ($1, true) ON CONFLICT (version) DO UPDATE SET dirty = true`, m.version)
		if err != nil {
			return fmt.Errorf("mark dirty %d: %w", m.version, err)
		}

		// Apply migration
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply version %d (%s): %w", m.version, m.name, err)
		}

		// Mark as clean
		_, err = db.Exec(`UPDATE schema_migrations SET dirty = false WHERE version = $1`, m.version)
		if err != nil {
			return fmt.Errorf("mark clean %d: %w", m.version, err)
		}

		fmt.Printf("  ✓ Applied: %06d_%s\n", m.version, m.name)
		applied++
	}

	if applied == 0 {
		fmt.Println("  No pending migrations")
	} else {
		fmt.Printf("  %d migrations applied\n", applied)
	}
	return nil
}

func migrateDown(db *sql.DB, dir string) error {
	migrations, err := loadMigrations(dir)
	if err != nil {
		return err
	}

	current, _ := currentVersion(db)
	if current == 0 {
		fmt.Println("  No migrations to rollback")
		return nil
	}

	// Find migration matching current version
	var target *migration
	for i := range migrations {
		if migrations[i].version == current {
			target = &migrations[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("no migration found for version %d", current)
	}
	if target.downFile == "" {
		return fmt.Errorf("no down file for version %d", current)
	}

	content, err := os.ReadFile(target.downFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", target.downFile, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("rollback version %d: %w", current, err)
	}

	_, err = db.Exec(`DELETE FROM schema_migrations WHERE version = $1`, current)
	if err != nil {
		return fmt.Errorf("delete version %d: %w", current, err)
	}

	fmt.Printf("  ✓ Rolled back: %06d_%s\n", target.version, target.name)
	return nil
}

func forceVersion(db *sql.DB, v int) error {
	// Clear all versions and set the forced one
	_, err := db.Exec(`DELETE FROM schema_migrations`)
	if err != nil {
		return err
	}
	if v > 0 {
		_, err = db.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES ($1, false)`, v)
	}
	return err
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
