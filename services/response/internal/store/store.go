// Package store persists response actions to Postgres.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Action is a response action record stored in the database.
type Action struct {
	ActionID    string
	TenantID    string
	IncidentID  string // may be empty
	AlertID     string // may be empty
	ActionType  string
	Target      string
	Status      string // "pending" | "success" | "failed"
	Reason      string
	TriggeredBy string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Detail      string // JSON
}

// Store wraps a pgxpool for response action persistence.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Insert writes a new action row.
func (s *Store) Insert(ctx context.Context, a *Action) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO response_actions
			(action_id, tenant_id, incident_id, alert_id, action_type,
			 target, status, reason, triggered_by, created_at, updated_at, detail)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12::jsonb)
		ON CONFLICT (tenant_id, action_id) DO NOTHING`,
		a.ActionID, a.TenantID, a.IncidentID, a.AlertID, a.ActionType,
		a.Target, a.Status, a.Reason, a.TriggeredBy,
		a.CreatedAt, a.UpdatedAt, a.Detail,
	)
	if err != nil {
		return fmt.Errorf("store: insert action: %w", err)
	}
	return nil
}

// UpdateStatus updates status and updated_at for an action.
func (s *Store) UpdateStatus(ctx context.Context, tenantID, actionID, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE response_actions
		   SET status = $3, updated_at = now()
		 WHERE tenant_id = $1 AND action_id = $2`,
		tenantID, actionID, status,
	)
	if err != nil {
		return fmt.Errorf("store: update status: %w", err)
	}
	return nil
}

// List returns the most recent N actions for a tenant.
func (s *Store) List(ctx context.Context, tenantID string, limit int) ([]*Action, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT action_id, tenant_id,
		       coalesce(incident_id,''), coalesce(alert_id,''),
		       action_type, target, status, reason, triggered_by,
		       created_at, updated_at, detail::text
		  FROM response_actions
		 WHERE tenant_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list actions: %w", err)
	}
	defer rows.Close()

	var actions []*Action
	for rows.Next() {
		var a Action
		if err := rows.Scan(
			&a.ActionID, &a.TenantID, &a.IncidentID, &a.AlertID,
			&a.ActionType, &a.Target, &a.Status, &a.Reason, &a.TriggeredBy,
			&a.CreatedAt, &a.UpdatedAt, &a.Detail,
		); err != nil {
			return nil, fmt.Errorf("store: scan: %w", err)
		}
		actions = append(actions, &a)
	}
	return actions, rows.Err()
}
