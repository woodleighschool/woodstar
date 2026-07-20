-- +goose Up

ALTER TABLE users
    ADD CONSTRAINT users_email_lowercase_check CHECK (email = lower(email)),
    ADD CONSTRAINT users_upn_lowercase_check CHECK (
        user_principal_name IS NULL OR user_principal_name = lower(user_principal_name)
    );
