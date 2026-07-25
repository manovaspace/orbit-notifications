package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manovaspace/orbit-notifications/internal/domain"
)

// Repository implements domain.DeliveryRepository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, record domain.DeliveryRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO delivery_records (id, template, channel, recipient_hash, status, dev_payload, correlation_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8::timestamptz, NOW()))
	`, record.ID, record.Template, record.Channel, record.RecipientHash, record.Status,
		nullJSON(record.DevPayload), record.CorrelationID, record.CreatedAt)
	return err
}

func (r *Repository) List(ctx context.Context, limit int, channel string) ([]domain.DeliveryRecord, error) {
	query := `
		SELECT id, template, channel, recipient_hash, status,
		       COALESCE(dev_payload::text, ''), COALESCE(correlation_id, ''), created_at
		FROM delivery_records
	`
	args := []any{limit}
	if channel != "" {
		query += ` WHERE channel = $2`
		args = append(args, channel)
	}
	query += ` ORDER BY created_at DESC LIMIT $1`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.DeliveryRecord
	for rows.Next() {
		var rec domain.DeliveryRecord
		if err := rows.Scan(&rec.ID, &rec.Template, &rec.Channel, &rec.RecipientHash,
			&rec.Status, &rec.DevPayload, &rec.CorrelationID, &rec.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func nullJSON(s string) any {
	if s == "" {
		return nil
	}
	return []byte(s)
}

// Migrate runs goose migrations from dir.
func Migrate(ctx context.Context, databaseURL, migrationsDir string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("postgres migrate connect: %w", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS goose_db_version (
		id serial PRIMARY KEY,
		version_id bigint NOT NULL,
		is_applied bool NOT NULL,
		tstamp timestamptz DEFAULT NOW()
	)`); err != nil {
		return err
	}

	sql, err := readMigration(migrationsDir + "/001_delivery_records.sql")
	if err != nil {
		return err
	}
	// ponytail: inline up migration for bootstrap without goose CLI in container
	const marker = "-- +goose Up"
	idx := indexOf(sql, marker)
	if idx < 0 {
		return fmt.Errorf("migration marker not found")
	}
	up := sql[idx+len(marker):]
	if downIdx := indexOf(up, "-- +goose Down"); downIdx >= 0 {
		up = up[:downIdx]
	}
	_, err = pool.Exec(ctx, up)
	return err
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func readMigration(path string) (string, error) {
	b, err := osReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
