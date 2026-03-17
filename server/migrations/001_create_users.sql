CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    device_token TEXT,
    department TEXT NOT NULL CHECK (department IN ('CSE', 'ECE', 'ME', 'CIVIL', 'EEE')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_department
    ON users (department);
