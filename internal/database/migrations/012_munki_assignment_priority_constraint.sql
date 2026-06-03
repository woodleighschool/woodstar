-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'munki_assignments'::regclass
          AND conname = 'munki_assignments_priority_check'
    ) THEN
        ALTER TABLE munki_assignments
            ADD CONSTRAINT munki_assignments_priority_check CHECK (priority >= 1);
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

ALTER TABLE munki_assignments
    DROP CONSTRAINT IF EXISTS munki_assignments_priority_check;
