package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"signalhub/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) StoreRawEvent(ctx context.Context, event *domain.RawEvent) error {
	query := `
		INSERT INTO raw_events (source, payload, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query, event.Source, event.Payload, event.CreatedAt).Scan(&event.ID)
	if err != nil {
		return fmt.Errorf("failed to store raw event: %w", err)
	}
	return nil
}

func (r *Repository) UpsertAlert(ctx context.Context, alert *domain.Alert) error {
	labels, err := json.Marshal(alert.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}
	annotations, err := json.Marshal(alert.Annotations)
	if err != nil {
		return fmt.Errorf("failed to marshal annotations: %w", err)
	}

	query := `
		INSERT INTO alerts (
			fingerprint, status, severity, environment, service, resource, team, 
			summary, description, labels, annotations, starts_at, ends_at, last_seen, occurrence_count, updated_at, source
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (fingerprint) DO UPDATE SET
			status = EXCLUDED.status,
			severity = EXCLUDED.severity,
			environment = EXCLUDED.environment,
			service = EXCLUDED.service,
			resource = EXCLUDED.resource,
			team = EXCLUDED.team,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			starts_at = EXCLUDED.starts_at,
			ends_at = EXCLUDED.ends_at,
			last_seen = EXCLUDED.last_seen,
			occurrence_count = alerts.occurrence_count + 1,
			updated_at = EXCLUDED.updated_at,
			source = EXCLUDED.source
		RETURNING id, created_at, occurrence_count
	`
	err = r.pool.QueryRow(ctx, query,
		alert.Fingerprint,
		alert.Status,
		alert.Severity,
		alert.Environment,
		alert.Service,
		alert.Resource,
		alert.Team,
		alert.Summary,
		alert.Description,
		labels,
		annotations,
		alert.StartsAt,
		alert.EndsAt,
		alert.LastSeen,
		alert.OccurrenceCount,
		time.Now(),
		alert.Source,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.OccurrenceCount)

	if err != nil {
		return fmt.Errorf("failed to upsert alert: %w", err)
	}
	return nil
}

func (r *Repository) AppendAlertEvent(ctx context.Context, event *domain.AlertEvent) error {
	query := `
		INSERT INTO alert_events (alert_id, status, event_time, details, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		event.AlertID,
		event.Status,
		event.EventTime,
		event.Details,
		event.CreatedAt,
	).Scan(&event.ID)

	if err != nil {
		return fmt.Errorf("failed to append alert event: %w", err)
	}
	return nil
}
