CREATE TABLE deliveries (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    notification_id UUID REFERENCES notifications(id),
    status TEXT,
    retry_count INT DEFAULT 0,
    delivered_at TIMESTAMP
);