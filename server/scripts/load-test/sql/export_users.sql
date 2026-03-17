SELECT COALESCE(json_agg(id::text ORDER BY created_at), '[]'::json)
FROM users;
