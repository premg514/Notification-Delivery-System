CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    device_token TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);