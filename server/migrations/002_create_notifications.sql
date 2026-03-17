CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL CHECK (length(btrim(title)) > 0),
    message TEXT NOT NULL CHECK (length(btrim(message)) > 0),
    target_department TEXT NOT NULL CHECK (target_department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE')),
    priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high')),
    idempotency_key TEXT UNIQUE CHECK (idempotency_key IS NULL OR length(btrim(idempotency_key)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_created_at
    ON notifications (created_at DESC);
