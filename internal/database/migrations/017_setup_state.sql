-- +goose Up

CREATE TABLE setup_state (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
    completed_at TIMESTAMPTZ
);

INSERT INTO setup_state (completed_at)
SELECT CASE
    WHEN EXISTS (
        SELECT 1
        FROM users
        WHERE role = 'admin'
          AND deleted_at IS NULL
    ) THEN now()
    ELSE NULL
END;
