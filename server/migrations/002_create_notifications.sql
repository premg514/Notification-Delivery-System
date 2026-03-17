CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    target_department TEXT NOT NULL CHECK (target_department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE')),
    priority TEXT NOT NULL DEFAULT 'normal',
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
