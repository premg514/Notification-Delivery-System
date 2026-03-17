ALTER TABLE users
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS department TEXT;

UPDATE users
SET department = 'CSE'
WHERE department IS NULL;

ALTER TABLE users
    ALTER COLUMN department SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_department_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_department_check
            CHECK (department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE'));
    END IF;
END $$;

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS target_department TEXT,
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT UNIQUE;

UPDATE notifications
SET target_department = 'CSE'
WHERE target_department IS NULL;

ALTER TABLE notifications
    ALTER COLUMN target_department SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'notifications_target_department_check'
    ) THEN
        ALTER TABLE notifications
            ADD CONSTRAINT notifications_target_department_check
            CHECK (target_department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE'));
    END IF;
END $$;

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

CREATE INDEX IF NOT EXISTS idx_users_department
    ON users (department);
