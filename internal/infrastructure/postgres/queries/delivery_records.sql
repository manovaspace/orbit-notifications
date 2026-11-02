-- name: InsertDelivery :exec
INSERT INTO delivery_records (
    id,
    template,
    channel,
    recipient_hash,
    status,
    dev_payload,
    correlation_id,
    created_at
) VALUES (
    sqlc.arg('id')::uuid,
    sqlc.arg('template'),
    sqlc.arg('channel'),
    sqlc.arg('recipient_hash'),
    sqlc.arg('status'),
    sqlc.narg('dev_payload')::text::jsonb,
    sqlc.arg('correlation_id')::text,
    COALESCE(sqlc.narg('created_at')::timestamptz, NOW())
);

-- name: ListDeliveries :many
SELECT
    id::text AS id,
    template,
    channel,
    recipient_hash,
    status,
    COALESCE(dev_payload::text, '')::text AS dev_payload,
    COALESCE(correlation_id, '')::text AS correlation_id,
    created_at
FROM delivery_records
WHERE (sqlc.narg('channel')::text IS NULL OR channel = sqlc.narg('channel')::text)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit')::int;
