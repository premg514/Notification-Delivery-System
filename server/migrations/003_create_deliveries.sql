CREATE TABLE IF NOT EXISTS deliveries (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    delivered_at TIMESTAMPTZ,
    last_error TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deliveries_notification_user
    ON deliveries (notification_id, user_id);

CREATE INDEX IF NOT EXISTS idx_deliveries_status_retry
    ON deliveries (status, retry_count);

CREATE INDEX IF NOT EXISTS idx_deliveries_user_id
    ON deliveries (user_id);
