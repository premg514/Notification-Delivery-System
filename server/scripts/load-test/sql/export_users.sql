SELECT COALESCE(json_agg(department ORDER BY department), '[]'::json)
FROM (
    SELECT DISTINCT department
    FROM users
) departments;
