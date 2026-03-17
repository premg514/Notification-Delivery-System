CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE CHECK (length(btrim(email)) > 0),
    device_token TEXT NOT NULL CHECK (length(btrim(device_token)) > 0),
    department TEXT NOT NULL CHECK (department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_department
    ON users (department, created_at, id);
