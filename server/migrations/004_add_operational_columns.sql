ALTER TABLE users
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT UNIQUE;

ALTER TABLE deliveries
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deliveries_notification_user
    ON deliveries (notification_id, user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_created_at
    ON notifications (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_idempotency_key
    ON notifications (idempotency_key);

CREATE INDEX IF NOT EXISTS idx_deliveries_status_retry
    ON deliveries (status, retry_count);

CREATE INDEX IF NOT EXISTS idx_deliveries_user_id
    ON deliveries (user_id);
