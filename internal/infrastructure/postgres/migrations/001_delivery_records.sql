-- +goose Up
-- Adoption: the previous custom runner applied this file without recording a
-- version row. Every statement is IF NOT EXISTS, so running it on a database
-- that already has these objects is a no-op. Do not remove the guards.
CREATE TABLE IF NOT EXISTS delivery_records (
    id UUID PRIMARY KEY,
    template TEXT NOT NULL,
    channel TEXT NOT NULL,
    recipient_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    dev_payload JSONB,
    correlation_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS delivery_records_created_at_idx ON delivery_records (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS delivery_records;
